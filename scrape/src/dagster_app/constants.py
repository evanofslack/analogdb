AWS_BUCKET_PHOTOS = "analog-photos"
AWS_BUCKET_TEST = "analog-photos-test"
AWS_BUCKET_COMMENTS = "analog-comments"

# Define subreddit names
ANALOG_SUB = "analog"
BW_SUB = "analog_bw"
SPROCKET_SUB = "SprocketShots"

# define number of posts to scrape per subreddit
ANALOG_POSTS = 10  # only scrapes 8 since 2 pinned self posts
BW_POSTS = 2  # only scrapes 2 since 1 pinned self post
SPROCKET_POSTS = 1

BLACKLIST_PATH = "data/keyword_blacklist.txt"
FILMS_PATH = "data/films.json"
CAMERAS_PATH = "data/cameras.json"

# maximum number of keywords to store in DB
KEYWORD_LIMIT = 50
# only update keywords for posts older than 2 days
KEYWORD_UPDATE_CUTOFF_DAYS = 2
