# Analog scraper

Python data pipeline providing stream of content for [AnalogDB](https://analogdb.herokuapp.com/) 

* Scrapes photos from [r/analog](https://www.reddit.com/r/analog/)
* Preforms image processing and resizing to generate variable resolution images
* Uploads to AWS S3 bucket, which serves photos behind CloudFront CDN
* Loads image metadata and urls to Postgres to be served by [AnalogDB API](https://github.com/evanofslack/analogdb)
* Runs every 8 hours with Heroku scheduler


