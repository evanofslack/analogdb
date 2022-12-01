AWS_BUCKET = "analog-photos"
AWS_BUCKET_TEST = "analog-photos-test"

# Define subreddit names
ANALOG_SUB = "analog"
BW_SUB = "analog_bw"
SPROCKET_SUB = "SprocketShots"

# define number of posts to scrape per subreddit
ANALOG_POSTS = 8  # only scrapes 6 since 2 pinned self posts
BW_POSTS = 1
SPROCKET_POSTS = 1

# define resolutions for resizing images
LOW_RES = (720, 720)
MEDIUM_RES = (1080, 1080)
HIGH_RES = (1440, 1440)
RAW_RES = None

# cloudfront base url
CLOUDFRONT_URL = "https://d3i73ktnzbi69i.cloudfront.net"

# analogdb base url
ANALOGDB_URL = "https://api.analogdb.com"

# reddit base url
REDDIT_URL = "https://www.reddit.com"

# valid media types
VALID_CONTENT = [
    "image/png",
    "image/jpeg",
    "image/jpg",
    "image/gif",
]
