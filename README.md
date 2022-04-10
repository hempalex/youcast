# youcast
**youcast** is a simple command line tool which downloads m4a audio files from youtube channel and makes RSS feed to use them as a podcast feed

It uses yt-dlp for downloading/extracting audio files from youtube and creates RSS file. You can use any web-server app to deliver podcasts to your devices.

## Getting Started

Usage:

```
youcast https://www.youtube.com/channel/<channel id> <podcast name> <base url> [cover url]
```

### Dependencies

* Linux/FreeBSD/MacOS
* Go compiler
* [yt-dlp](https://github.com/yt-dlp/yt-dlp)
* ffmpeg
* mutagen/AtomicParsley tool for embedding metadata
* web-server like nginx or apache

### Installing

* use your operating system packaging tool to install go/yt-dlp/ffmpeg/nginx
* clone this repository 
* run **make binary** or **make release** (to build for all supported systems)
* copy binary to appropriate place, like /opt/youcast/bin

### Usage

First create directory for your podcasts storage, for example /var/www/podcasts
Then create shell script to run youcast, for example /opt/bin/youcast.sh

```
#!/bin/sh

cd /var/www/podcasts
/opt/bin/youcast https://www.youtube.com/channel/<channel id>/videos <channel keyword> <base url> <url to channel artwork>
```

Where:

`<channel id>` - youtube channel id (you could just click on "video" section of your favorite youtube channel and copy url)

`<channel keyword>` - a filename for rss feed

`<base url>` - a url pointing to podcasts directory, like http://www.site.tld/podcasts  

`<url to channel artwork>` - you could copy a url to channel image from youtube channel page and paste it here

Don't forget **chnmod 0755 /opt/bin/youcast.sh** and run it for testing
For more than one channel just add a call to youcast for each channel url

Run it and in podcasts directory (/var/www/podcasts) you should see a  `<channel_keyword>.rss` file and a `audio` directory which holds `m4a` files.

Secondly make your web server config (example for nginx.conf)
 
```
server {
   server_name wwww.site.tld;
  
  # ....
  
  location /podcasts {
      alias /var/www/podcasts;
      autoindex on;
  }
  
  # ....
  
}
```

Then reload your webserver config (nginx -t && nginx -s reload) and open http://wwww.site.tld/podcasts in browser. 

You should see a list of RSS files. 
Copy url of one of then and use it to add to your favorite podcast catcher application. 
I prefer `Downcast` and `Overcast` for iOS but RSS shoud be compatible to almost any podcast app.
