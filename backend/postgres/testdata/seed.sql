-- seed.sql

-- Insert test pictures
INSERT INTO pictures (
    url, title, author, permalink, description, score, nsfw, greyscale, time, width, height, sprocket,
    lowurl, lowwidth, lowheight, medurl, medwidth, medheight, highurl, highwidth, highheight,
    camera_make, camera_model, film_make, film_type, film_speed, focal_length, aperture
) VALUES 
(
    'https://example.com/raw1.jpg',
    'Sunset Photography',
    'u/photographer1',
    'reddit.com/sunset_photo_123',
    'description of a sunset',
    150,
    false,
    false,
    1640995200,
    3000,
    2000,
    false,
    'https://example.com/low1.jpg',
    300,
    200,
    'https://example.com/med1.jpg',
    800,
    533,
    'https://example.com/high1.jpg',
    1600,
    1067,
    'Canon',
    'AE-1',
    'Kodak',
    'Portra 400',
    400,
    50,
    'f/2.8'
),
(
    'https://example.com/raw2.jpg',
    'Street Scene',
    'u/streetphotographer',
    'reddit.com/street_scene_456',
    '',
    85,
    false,
    true,
    1641081600,
    2400,
    3600,
    true,
    'https://example.com/low2.jpg',
    240,
    360,
    'https://example.com/med2.jpg',
    600,
    900,
    'https://example.com/high2.jpg',
    1200,
    1800,
    'Nikon',
    'FM2',
    'Ilford',
    'HP5 Plus',
    400,
    35,
    'f/5.6'
),
(
    'https://example.com/raw3.jpg',
    'Portrait Study',
    'u/portraitist',
    'reddit.com/portrait_study_789',
    'portrait studies are my favorite',
    220,
    true,
    false,
    1641168000,
    2000,
    3000,
    false,
    'https://example.com/low3.jpg',
    200,
    300,
    'https://example.com/med3.jpg',
    500,
    750,
    'https://example.com/high3.jpg',
    1000,
    1500,
    'Pentax',
    'K1000',
    'Fuji',
    'Pro 400H',
    400,
    85,
    'f/1.4'
);

-- Insert test keywords
INSERT INTO keywords (word, weight, post_id) VALUES
('sunset', 0.95, 1),
('landscape', 0.85, 1),
('golden hour', 0.75, 1),
('street', 0.90, 2),
('black and white', 0.88, 2),
('urban', 0.70, 2),
('portrait', 0.95, 3),
('studio', 0.80, 3),
('lighting', 0.75, 3);

-- Insert test colors
INSERT INTO colors (hex, css, html, percent, post_id) VALUES
('#FF6B35', 'rgb(255, 107, 53)', 'orange', 0.35, 1),
('#F7931E', 'rgb(247, 147, 30)', 'darkorange', 0.28, 1),
('#FFD23F', 'rgb(255, 210, 63)', 'gold', 0.20, 1),
('#06FFA5', 'rgb(6, 255, 165)', 'springgreen', 0.12, 1),
('#4D4D4D', 'rgb(77, 77, 77)', 'dimgray', 0.05, 1),
('#2C2C2C', 'rgb(44, 44, 44)', 'darkgray', 0.45, 2),
('#5A5A5A', 'rgb(90, 90, 90)', 'gray', 0.30, 2),
('#808080', 'rgb(128, 128, 128)', 'gray', 0.15, 2),
('#A0A0A0', 'rgb(160, 160, 160)', 'darkgray', 0.07, 2),
('#CCCCCC', 'rgb(204, 204, 204)', 'lightgray', 0.03, 2),
('#FFB6C1', 'rgb(255, 182, 193)', 'lightpink', 0.40, 3),
('#FFC0CB', 'rgb(255, 192, 203)', 'pink', 0.25, 3),
('#FFEFD5', 'rgb(255, 239, 213)', 'papayawhip', 0.20, 3),
('#F5DEB3', 'rgb(245, 222, 179)', 'wheat', 0.10, 3),
('#DEB887', 'rgb(222, 184, 135)', 'burlywood', 0.05, 3);

-- Insert test post updates
INSERT INTO post_updates (
    post_id, score_update_time, nsfw_update_time, greyscale_update_time,
    sprocket_update_time, colors_update_time, keywords_update_time
) VALUES
(1, 1641254400, null, null, null, 1641254400, 1641254400),
(2, 1641340800, 1641340800, 1641340800, 1641340800, 1641340800, 1641340800),
(3, 1641427200, 1641427200, null, null, 1641427200, 1641427200);

INSERT INTO films (film_make, film_type, film_speed, color_type, description) VALUES
('fuji', 'acros', 100, 'bw', 'Fine grain black and white'),
('kodak', 'portra', 400, 'color', 'Professional color negative'),
('kodak', 'tri-x', 400, 'bw', 'Classic black and white');


INSERT INTO cameras (camera_make, camera_model, description) VALUES
('canon', 'ae-1', 'Classic 35mm SLR camera'),
('leica', 'm6', 'Premium rangefinder camera'),
('nikon', 'fm2', 'Mechanical SLR with electronic shutter');
