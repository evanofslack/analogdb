# scraper

python data pipeline providing stream of content for [AnalogDB](https://analogdb.com)

### features

- scrapes posts from [r/analog](https://www.reddit.com/r/analog/)
- performs image processing, resizing and primary color extraction
- identifies keywords from post comments
- uploads images and comments to AWS S3 bucket
