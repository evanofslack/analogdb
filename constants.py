AWS_BUCKET = "analog-photos"
AWS_BUCKET_TEST = "analog-photos-test"

# Define subreddit names
ANALOG_SUB = "analog"
BW_SUB = "analog_bw"
SPROCKET_SUB = "SprocketShots"

# define number of posts to scrape per subreddit
ANALOG_POSTS = 10  # only scrapes 8 since 2 pinned self posts
BW_POSTS = 2  # only scrapes 2 since 1 pinned self post
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
# ANALOGDB_URL = "http://10.33.1.142:8080"

# reddit base url
REDDIT_URL = "https://www.reddit.com"

# valid media types
VALID_CONTENT = [
    "image/png",
    "image/jpeg",
    "image/jpg",
    "image/gif",
]

# Upper limit to the number of extracted
# colors presented in the output.
COLOR_LIMIT = 5

# Group colors to limit the output and give a
# better visual representation. Based on a
# scale from 0 to 100. Where 0 won't group any
# color and 100 will group all colors into one.
# Tolerance 0 will bypass all conversion.
COLOR_TOLERANCE = 20
