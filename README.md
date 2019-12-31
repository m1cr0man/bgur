# bgur - Desktop backgrounds without the effort

Have you got a massive collection of desktop backgrounds favourited
on [imgur](https://imgur.com/)? Time to throw them into a folder and
let bgur do the magic to set them as your background.

## Features

- Set desktop background from imgur (you'd hope so at least. Macs untested)
- Randomise backgrounds with anti-repeat logic
- minratio + maxratio options to ignore mobile oriented photos on desktop
- Syncing! Uses imgur, an album, and your own account - so no GDPR shenanigans
- Caching so that it doesn't kill imgur (offline coming soon)

## Usage

- If you don't have one already, set up a favourites folder on imgur with all
your desktop backgrounds. Call it `Desktop Backgrounds` (optional). It can be
a mix of private images, public images and albums.
- Alternatively, you can use a friend's folder. See the `-folder-owner` and
`-folder-name` options
- Install and run bgur
```bash
go get github.com/m1cr0man/bgur
go build -o bgur github.com/m1cr0man/bgur/cmd/bgur
chmod +x bgur # linux users
```
- Run bgur (basic usage)
```bash
./bgur -sync
``` 

## Advanced usage

You can run `bgur -h` to get all the options correct to the version you installed.
I have to update the below list manually, but if you are lazy you can reference this.

```bash
Usage of ./bgur:
  -change-interval int
        Minutes between background changes. Default is 12 hours (default 720)
  -folder-name string
        Name of the folder to pull desktop backgrounds from (default "desktop backgrounds")
  -folder-owner string
        Username who owns the backgrounds folder. Defaults to you
  -force-change
        Force a background change now. Overrides expiry
  -max-ratio int
        Maximum ratio of width:height, in percent. Use this for vertical screens, overrides minRatio
  -min-ratio int
        Minimum ratio of width:height, in percent. For example 160 which is 16:10
  -refresh-cache
        Refresh list of images from the folder on Imgur
  -seed int
        Seed to use for shuffling the folder. Set to 0 to skip shuffling (default 1577751173)
  -sync
        Sync state to Imgur so that the same backgrounds appear on other computers
```

## TODO

- Auto building of the project
- Logo
- Work offline properly
- A web UI, because not everyone is a CLI hero. This will not be an electron app.
