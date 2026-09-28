import analogdb.models as analog
import scrape.models as scrape


def convert_create(p: scrape.CreatePost) -> analog.PostCreate:
    return analog.PostCreate(
        title=p.title,
        author=p.author,
        permalink=p.permalink,
        description=p.description,
        score=p.score,
        nsfw=p.nsfw,
        grayscale=p.grayscale,
        timestamp=p.time,
        sprocket=p.sprocket,
        camera_make=p.camera_make,
        camera_model=p.camera_model,
        film_make=p.film_make,
        film_type=p.film_type,
        film_speed=p.film_speed,
        focal_length=p.focal_length,
        aperture=p.aperture,
        images=[convert_image(i) for i in p.images],
        keywords=[convert_keyword(k) for k in p.keywords],
        colors=[convert_color(c) for c in p.colors],
    )


def convert_image(i: scrape.S3Image) -> analog.Image:
    return analog.Image(
        url=i.url, resolution=i.resolution, width=i.width, height=i.height
    )


def convert_color(c: scrape.Color) -> analog.Color:
    return analog.Color(hex=c.hex, css=c.css, html=c.html, percent=c.percent)


def convert_keyword(k: scrape.Keyword) -> analog.Keyword:
    return analog.Keyword(word=k.word, weight=k.weight)
