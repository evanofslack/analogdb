CREATE TABLE IF NOT EXISTS pictures (
  id SERIAL PRIMARY KEY,
  url text NOT NULL UNIQUE   
  title text, 
  author text,
  permalink text,
  score integer,
  nsfw boolean,
  greyscale boolean,
  time integer,
  width integer,
  height integer,
  sprocket bool,
  lowURL string,
  lowWidth int,
  lowHeight int,
  medUrl string,
  medWidth int,
  medHeight int,
  highUrl string,
  highWidth int,
  highHeight int,
);

INSERT 
INTO pictures(url, title, author, permalink, score, nsfw, greyscale, time, width, height, sprocket, lowUrl, lowWidth, lowHeight, medUrl, medWidth, medHeight, highUrl, highWidth, highHeight) 
VALUES (www.google.com, testTitle, testAuthor, www.permalink.com, 69, true, false, 1001001001, 10, 20, false, www.lowurl.com, 1, 2, www.mediumurl.com,2, 4, www.highurl.com, 5, 10) 
ON CONFLICT (permalink) DO NOTHING