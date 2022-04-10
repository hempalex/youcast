package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const rssDateFormat = "Mon, 2 Jan 2006 15:04:05 -0700"
const xmlDateFormat = "2006-01-02T15:04:05-07:00"

// YouCast represents podcast feed
type YouCast struct {
	id  string
	url string
}

type videoFeed struct {
	Name     string   // name for podcast rss file
	BaseURL  string   // baseurl for audio links
	ImageURL string   // url for podcast artwork
	XMLName  xml.Name `xml:"feed"`
	Link     struct {
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
	} `xml:"link"`
	ID        string `xml:"id"`
	ChannelID string `xml:"channelId"`
	Title     string `xml:"title"`
	Author    struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Published string `xml:"published"`
	Build     string
	Entry     []*struct {
		Duration string
		Length   int64
		ID       string `xml:"videoId"`
		Title    string `xml:"title"`
		Link     struct {
			Rel  string `xml:"rel,attr"`
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Author struct {
			Name string `xml:"name"`
		} `xml:"author"`
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
		Group     struct {
			Title     string `xml:"title"`
			Thumbnail struct {
				URL string `xml:"url,attr"`
			} `xml:"thumbnail"`
			Description string `xml:"description"`
		} `xml:"group"`
	} `xml:"entry"`
}

var rssTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>{{.Title}}</title>
	<description>{{.Title}}</description>
	<link>{{.BaseURL}}</link>
    <category>Unknown</category>
    <generator>youcast</generator>
    <language>en-us</language>
    <lastBuildDate>{{.Build}}</lastBuildDate>
	<pubDate>{{.Published}}</pubDate>
	{{if ne .ImageURL "" }}<itunes:image href="{{.ImageURL}}" />{{end}}
  <itunes:author>{{.Author.Name}}</itunes:author>
	{{range .Entry}}
		{{if gt .Length 0 }}
		<item>
			<guid isPermaLink="false">https://www.youtube.com/v/{{.ID}}</guid>
			<title>{{.Title}}</title>
			<link>https://www.youtube.com/watch?v={{.ID}}</link>
			<pubDate>{{.Published}}</pubDate>
			<description><![CDATA[{{.Group.Description}}]]></description>
			<enclosure url="{{$.BaseURL}}/audio/{{$.Name}}/{{.ID}}.m4a" length="{{.Length}}" type="audio/mp4"></enclosure>
			<itunes:duration>{{.Duration}}</itunes:duration>
			<itunes:image>{{.Group.Thumbnail.URL}}</itunes:image>
		</item>
		{{end}}
	{{end}}
  </channel>
</rss>`

var opmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
    <head>
        <title>{{.Title}}</title>
    </head>
    <body>
        <outline text="{{.Title}}" type="rss" title="{{.Title}}" xmlUrl="{{.BaseURL}}/{{.Name}}.rss" htmlUrl="{{.Link.Href}}"/>
    </body>
</opml>
`

// New creates new structure
func New(URL string) (*YouCast, error) {

	reg := regexp.MustCompile(`https://www.youtube.com/channel/([\w_-]+)`)
	res := reg.FindStringSubmatch(URL)

	if res == nil {
		return nil, fmt.Errorf("%s does not contain youtube channel url", URL)
	}

	return &YouCast{id: res[1], url: URL}, nil
}

func (cast *YouCast) String() string {
	return cast.id
}

// XMLURL returns an url to channel xml feed
func (cast *YouCast) XMLURL() string {
	return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", cast.id)
}

func ytISODateToRFC822(date string) (string, error) {
	dt, err := time.Parse(xmlDateFormat, date)
	if err != nil {
		return "", err
	}
	return dt.Format(rssDateFormat), nil
}

func formatDurationHMS(d time.Duration) string {
	h := int64(d.Hours()) // часы могут быть больше суток
	m := int64(d.Minutes()) % 60
	s := int64(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// Process feed
func (cast *YouCast) Process(name string, baseURL string, imageURL string) error {

	// parse RSS template
	var rssTpl = template.New("rss")
	if _, err := rssTpl.Parse(rssTemplate); err != nil {
		return err
	}

	// parse OPML template
	var opmlTpl = template.New("opml")
	if _, err := opmlTpl.Parse(opmlTemplate); err != nil {
		return err
	}

	// output feed structure
	var feed videoFeed
	feed.Name = name
	feed.BaseURL = baseURL
	feed.ImageURL = imageURL
	feed.Build = time.Now().Format(rssDateFormat)

	// get youtube channel xml
	text, err := cast.getXML()
	if err != nil {
		return err
	}

	if err = xml.Unmarshal(text, &feed); err != nil {
		return err
	}

	// change format of Published field
	if feed.Published, err = ytISODateToRFC822(feed.Published); err != nil {
		return err
	}

	// number of videos in channel xml
	num := len(feed.Entry)

	// creating a folder to keep audio files
	folder := fmt.Sprintf("audio/%s", name)
	if err = os.MkdirAll(folder, 0755); err != nil {
		return err
	}

	// constructing yt-dlp command line
	ytdl := exec.Command("yt-dlp", "-i", "--embed-thumbnail", "--add-metadata", "-f", "140", "-o", fmt.Sprintf("%s/%%(id)s.m4a", folder), "--playlist-end", fmt.Sprintf("%d", num), cast.url)
	ytdl.Stdout = os.Stdout
	ytdl.Stderr = os.Stderr
	// run command to download all audio files in m4a format
	ytdl.Run()

	//  IDs of existing audio files
	entryIDs := make(map[string]bool)

	for _, entry := range feed.Entry {

		entryIDs[entry.ID] = true

		// cnange date formatting
		if entry.Published, err = ytISODateToRFC822(entry.Published); err != nil {
			return err
		}

		// filename
		filename := fmt.Sprintf("%s/%s.m4a", folder, entry.ID)

		stat, err := os.Stat(filename)
		if err != nil {
			// file not exists
			fmt.Printf("File %s not found\n", filename)
			entry.Length = 0
			continue
		} else {
			entry.Length = stat.Size()
			fmt.Printf("File %s length %d\n", filename, entry.Length)
		}

		// command to check audio duration
		ffprobe := exec.Command("ffprobe", "-v", "quiet", "-of", "default=nokey=1:noprint_wrappers=1", "-select_streams", "a:0", "-show_entries", "stream=duration", filename)
		ffprobe.Stderr = os.Stderr

		// run it and parse the output
		out, err := ffprobe.Output()
		if err != nil {
			return err
		}
		res := strings.Split(string(out), "\n")
		duration, err := time.ParseDuration(fmt.Sprintf("%ss", res[0]))
		if err != nil {
			return err
		}

		// round duration to one second precision and format
		entry.Duration = formatDurationHMS(duration.Round(time.Second))

		fmt.Printf("%s %s\n", entry.ID, entry.Duration)

	}

	// remove non-existing in youtube feed files
	files, err := filepath.Glob(folder + "/*.m4a")
	if err != nil {
		return err
	}
	for _, file := range files {
		id := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		if _, ok := entryIDs[id]; !ok {
			fmt.Printf("Removing %s\n", file)
			if err := os.Remove(file); err != nil {
				return err
			}
		}
	}

	// creating RSS feed
	rssFileName := name + ".rss"
	var rssFile *os.File

	if rssFile, err = os.Create(rssFileName); err != nil {
		return err
	}
	defer rssFile.Close()

	rssBw := bufio.NewWriter(rssFile)
	defer rssBw.Flush()

	if err = rssTpl.Execute(rssBw, feed); err != nil {
		return err
	}

	// creating OPML file
	opmlFileName := name + ".opml"
	var opmlFile *os.File

	if opmlFile, err = os.Create(opmlFileName); err != nil {
		return err
	}
	defer opmlFile.Close()

	opmlBw := bufio.NewWriter(opmlFile)
	defer opmlBw.Flush()

	if err = opmlTpl.Execute(opmlBw, feed); err != nil {
		return err
	}

	return nil
}

// downloads youtube XML feed
func (cast *YouCast) getXML() ([]byte, error) {

	url := cast.XMLURL()

	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("Status error: %v", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("Read body: %v", err)
	}

	return data, nil
}

func usage(err string) {
	if err == "" {
		fmt.Printf("Usage: %s https://www.youtube.com/channel/<channel id> <podcast name> <base url> [cover url]\n", os.Args[0])
	} else {
		fmt.Println(err)
	}
	os.Exit(1)
}

func main() {

	if len(os.Args) < 4 {
		usage("")
	}

	var imageURL string

	channelURL := os.Args[1]
	fmt.Printf("Channel URL: %s\n", channelURL)

	channel, err := New(channelURL)

	if err != nil {
		usage(err.Error())
	}

	fmt.Printf("Channel XML: %s\n", channel.XMLURL())

	if len(os.Args) > 4 {
		imageURL = os.Args[4]
	}

	err = channel.Process(os.Args[2], os.Args[3], imageURL)
	if err != nil {
		usage(err.Error())
	}

}
