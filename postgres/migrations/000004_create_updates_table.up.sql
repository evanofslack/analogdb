CREATE TABLE IF NOT EXISTS post_updates(
   id SERIAL PRIMARY KEY,
   post_id INT NOT NULL,
   score_update_time integer,
   nsfw_update_time integer,
   greyscale_update_time integer,
   sprocket_update_time integer,
   colors_update_time integer,
   keywords_update_time integer,
   CONSTRAINT fk_post_id
	   FOREIGN KEY(post_id)
		   REFERENCES pictures(id)
			   ON DELETE CASCADE
);
