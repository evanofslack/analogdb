AWS_BUCKET_PHOTOS = "analog-photos"
AWS_BUCKET_TEST = "analog-photos-test"
AWS_BUCKET_COMMENTS = "analog-comments"

# Define subreddit names
ANALOG_SUB = "analog"
BW_SUB = "analog_bw"
SPROCKET_SUB = "SprocketShots"

# define resolutions for resizing images
LOW_RES = (720, 720)
MEDIUM_RES = (1080, 1080)
HIGH_RES = (1440, 1440)
RAW_RES = None

# cloudfront base url
CLOUDFRONT_URL = "https://d3i73ktnzbi69i.cloudfront.net"

# reddit base url
REDDIT_URL = "https://www.reddit.com"

# valid media types
VALID_CONTENT = [
    "image/png",
    "image/jpeg",
    "image/jpg",
    "image/gif",
]

# connect and read timeouts in seconds for image downloads
IMAGE_TIMEOUT = (10, 60)

# jpeg quality for resized images
JPEG_QUALITY = 90

# max 99th percentile channel spread for an image to count as grayscale
GRAYSCALE_TOLERANCE = 15

# posts from these subreddits are always grayscale
GRAYSCALE_SUBREDDITS: set[str] = set()

# upper limit to the number of extracted
# colors presented in the output.
COLOR_LIMIT = 5

# group colors to limit the output and give a
# better visual representation. Based on a
# scale from 0 to 100. Where 0 won't group any
# color and 100 will group all colors into one.
# Tolerance 0 will bypass all conversion.
COLOR_TOLERANCE = 20
