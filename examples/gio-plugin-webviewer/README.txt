WEBVIEWER SHELL - Run Any Website as a Desktop App
===================================================

Quick Start (3 steps):

  1. Edit app.json
     Change "url" to your website address:

     {
         "url": "https://your-website.com",
         "name": "My App"
     }

  2. Double-click the app to launch

  3. Done! Your website runs as a native desktop app.


macOS Users - First Launch:
  macOS may show "app can't be opened" the first time.
  Fix: Right-click the app -> Open -> click "Open" in the dialog.

  Alternative: Open Terminal, navigate to this folder, and run:
    xattr -cr gio-plugin-webviewer.app


app.json Settings:
  url      Your website address (required)
  name     Window title (default: "Gio WebViewer")
  width    Window width in pixels (default: 1200)
  height   Window height in pixels (default: 800)

  Example:
  {
      "url": "https://my-app.example.com",
      "name": "My Cool App",
      "width": 1400,
      "height": 900
  }


Self-Update:
  The app can update itself from GitHub releases.
  To check for updates, open Terminal and run:
    ./gio-plugin-webviewer.app/Contents/MacOS/gio-plugin-webviewer --update

  On Windows:
    gio-plugin-webviewer.exe --update


Troubleshooting:
  Black screen?    Check that app.json has a valid URL
  Won't open?      macOS: right-click -> Open (see above)
  Wrong website?   Edit app.json and relaunch the app
  Need to resize?  Change width/height in app.json


More Info:
  https://github.com/joeblew999/utm-dev
