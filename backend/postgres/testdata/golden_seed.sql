-- extra posts for the findPosts golden test, loaded after seed.sql

INSERT INTO pictures (
    url, title, author, permalink, description, score, nsfw, greyscale, time, width, height, sprocket,
    lowurl, lowwidth, lowheight, medurl, medwidth, medheight, highurl, highwidth, highheight,
    camera_make, camera_model, film_make, film_type, film_speed, focal_length, aperture
) VALUES
('https://example.com/raw4.jpg', 'Beach, day [Nikon FM2 | Portra 400]', 'u/beachgoer', 'reddit.com/beach_day_4', NULL, 310, false, false, 1641254400, 4000, 2667, false,
 'https://example.com/low4.jpg', 720, 480, 'https://example.com/med4.jpg', 1080, 720, 'https://example.com/high4.jpg', 1440, 960,
 'nikon', 'fm2', 'kodak', 'portra 400', 400, 50, 'f/8'),
('https://example.com/raw5.jpg', 'Portrait without colors', 'u/portraitist', 'reddit.com/portrait_5', 'no colors here', 42, false, true, 1641340800, 2667, 4000, false,
 'https://example.com/low5.jpg', 480, 720, 'https://example.com/med5.jpg', 720, 1080, 'https://example.com/high5.jpg', 960, 1440,
 NULL, NULL, 'ilford', 'hp5 plus', 400, NULL, NULL),
('https://example.com/raw6.jpg', 'Harbor at dusk', 'u/b', 'reddit.com/harbor_6', '', 77, false, false, 1641427200, 3000, 3000, true,
 'https://example.com/low6.jpg', 720, 720, 'https://example.com/med6.jpg', 1080, 1080, 'https://example.com/high6.jpg', 1440, 1440,
 'mamiya', '7', NULL, NULL, NULL, 80, 'f/4'),
('https://example.com/raw7.jpg', 'Empty post', 'u/nobody', 'reddit.com/empty_7', NULL, 0, false, false, 1641513600, 1000, 1500, false,
 'https://example.com/low7.jpg', 480, 720, 'https://example.com/med7.jpg', 720, 1080, 'https://example.com/high7.jpg', 960, 1440,
 NULL, NULL, NULL, NULL, NULL, NULL, NULL),
('https://example.com/raw8.jpg', 'Night street portrait', 'u/streetphotographer', 'reddit.com/night_street_8', 'late night', 505, true, false, 1641600000, 4000, 2667, false,
 'https://example.com/low8.jpg', 720, 480, 'https://example.com/med8.jpg', 1080, 720, 'https://example.com/high8.jpg', 1440, 960,
 'leica', 'm6', 'cinestill', '800t', 800, 35, 'f/1.4');

INSERT INTO keywords (word, weight, post_id) VALUES
('beach', 0.91, 4),
('sand, sun', 0.62, 4),
('ocean', 0.33333333, 4),
('portrait', 0.87, 5),
('studio', 0.4, 5),
('portrait', 0.7, 8),
('street', 0.65, 8),
('night', 0.12345678, 8);

INSERT INTO colors (hex, css, html, percent, post_id) VALUES
('#f5deb3', 'wheat', 'tan', 0.41, 4),
('#4682b4', 'steelblue', 'teal', 0.27, 4),
('#ffffff', 'white', 'white', 0.18, 4),
('#808080', 'gray', 'gray', 0.09, 4),
('#000000', 'black', 'black', 0.05, 4),
('#008080', 'teal', 'teal', 0.55000000, 6),
('#000080', 'navy', 'navy', 0.2, 6),
('#696969', 'dimgray', 'gray', 0.12, 6),
('#a9a9a9', 'darkgray', 'gray', 0.08, 6),
('#dcdcdc', 'gainsboro', 'gray', 0.05, 6),
('#191970', 'midnightblue', 'navy', 0.6, 8),
('#000000', 'black', 'black', 0.25, 8),
('#8b0000', 'darkred', 'red', 0.1, 8),
('#ff7f50', 'coral', 'orange', 0.04, 8),
('#ffffff', 'white', 'white', 0.01, 8);

-- created and updated default to now(), pin them so the golden file is stable
UPDATE pictures SET created = to_timestamp(time), updated = to_timestamp(time);
