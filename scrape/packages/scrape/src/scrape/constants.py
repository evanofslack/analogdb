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

# imgur rejects the default python-requests user agent
USER_AGENT = "analogdb/1.0 (+https://analogdb.com)"

# hosts redirect deleted images here and serve a placeholder that is a valid image
REMOVED_IMAGE_URLS = {"i.imgur.com/removed.png"}

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

# k-means clusters fit in OKLab before merging and keeping the top COLOR_LIMIT
PALETTE_CLUSTERS = 8

# clusters closer than this OKLab distance are merged into one
PALETTE_MERGE_DISTANCE = 0.03

# max pixels sampled to fit the clusters, every pixel is still counted for shares
PALETTE_SAMPLE = 40000

# fixed seed so the same image always gives the same palette
PALETTE_SEED = 7
