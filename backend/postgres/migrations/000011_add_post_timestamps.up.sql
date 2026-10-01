ALTER TABLE pictures ADD COLUMN created timestamptz, ADD COLUMN updated timestamptz;

-- existing posts have no insert time, so use the reddit post time
UPDATE pictures SET
   created = COALESCE(to_timestamp(time), now()),
   updated = COALESCE(to_timestamp(time), now());

-- posts patched before now take their latest patch time
UPDATE pictures p
SET updated = GREATEST(p.created, to_timestamp(u.t))
FROM (
   SELECT post_id, max(GREATEST(score_update_time, nsfw_update_time, greyscale_update_time,
                                sprocket_update_time, colors_update_time, keywords_update_time)) AS t
   FROM post_updates
   GROUP BY post_id
) u
WHERE u.post_id = p.id AND u.t IS NOT NULL;

ALTER TABLE pictures
   ALTER COLUMN created SET DEFAULT now(),
   ALTER COLUMN created SET NOT NULL,
   ALTER COLUMN updated SET DEFAULT now(),
   ALTER COLUMN updated SET NOT NULL;
