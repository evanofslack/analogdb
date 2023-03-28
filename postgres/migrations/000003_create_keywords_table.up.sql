CREATE TABLE keywords(
id SERIAL PRIMARY KEY,
word VARCHAR(255) NOT NULL,
percent: NUMERIC(9, 8)
post_id INT
CONSTRAINT fk_post_id
	FOREIGN KEY(post_id)
	REFERENCES pictures(id)
)
