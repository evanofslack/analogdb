import analogdb.models as analog
import scrape.models as scrape


def convert_create(p: scrape.CreatePost) -> analog.PostCreate:
    return analog.PostCreate(
        p.title,
        p.author,
        p.permalink,
        p.score,
        p.nsfw,
        p.grayscale,
        p.time,
        p.sprocket,
        p.camera_make,
        p.camera_model,
        p.film_make,
        p.film_type,
        p.film_speed,
        p.focal_length,
        p.aperture,
        [convert_image(i) for i in p.images],
        [convert_keyword(k) for k in p.keywords],
        [convert_color(c) for c in p.colors],
    )


def convert_image(i: scrape.S3Image) -> analog.Image:
    return analog.Image(i.url, i.resolution, i.width, i.height)


def convert_color(c: scrape.Color) -> analog.Color:
    return analog.Color(c.hex, c.css, c.html, c.percent)


def convert_keyword(k: scrape.Keyword) -> analog.Keyword:
    return analog.Keyword(k.word, k.weight)
