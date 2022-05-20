CREATE TABLE IF NOT EXISTS "public"."pictures" (
    "id" integer NOT NULL,
    "url" "text" NOT NULL,
    "title" "text",
    "author" "text",
    "permalink" "text",
    "score" integer,
    "nsfw" boolean,
    "greyscale" boolean,
    "time" integer,
    "width" integer,
    "height" integer,
    "sprocket" boolean DEFAULT false,
    "lowurl" "text",
    "lowwidth" integer,
    "lowheight" integer,
    "medurl" "text",
    "medwidth" integer,
    "medheight" integer,
    "highurl" "text",
    "highwidth" integer,
    "highheight" integer
);

CREATE SEQUENCE "public"."pictures_id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE "public"."pictures_id_seq" OWNED BY "public"."pictures"."id";


ALTER TABLE ONLY "public"."pictures" ALTER COLUMN "id" SET DEFAULT "nextval"('"public"."pictures_id_seq"'::"regclass");


COPY "public"."pictures" ("id", "url", "title", "author", "permalink", "score", "nsfw", "greyscale", "time", "width", "height", "sprocket", "lowurl", "lowwidth", "lowheight", "medurl", "medwidth", "medheight", "highurl", "highwidth", "highheight") FROM '/var/lib/postgresql/csv/analog-data.csv' DELIMITER ',' CSV HEADER;


SELECT pg_catalog.setval('"public"."pictures_id_seq"', 2572, true);


ALTER TABLE ONLY "public"."pictures"
    ADD CONSTRAINT "pictures_pkey" PRIMARY KEY ("id");


ALTER TABLE ONLY "public"."pictures"
    ADD CONSTRAINT "pictures_url_key" UNIQUE ("url");


ALTER TABLE ONLY "public"."pictures"
    ADD CONSTRAINT "unique_url" UNIQUE ("permalink");
