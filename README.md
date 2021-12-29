# AnalogDB API

API serving film photographs scraped from [r/analog](https://www.reddit.com/r/analog/)

Built with [Go](https://go.dev/) and [chi](https://github.com/go-chi/chi)

Documentation for the API: https://analogdb.herokuapp.com/

Source code for scraper: https://github.com/evanofslack/analog-scraper

### Example

```bash
curl https://analogdb.herokuapp.com/latest/2
```

```yaml
{
    posts: [{
            "url": "https://i.redd.it/iar3s9niiw781.jpg",
            "title": "North Dakota | Mamiya 7ii 80MM | FujiFilm Pro 400H",
            "author": "u/26Point2",
            "permalink": "https://www.reddit.com/r/analog/comments/royi2l/north_dakota_mamiya_7ii_80mm_fujifilm_pro_400h/",
            "upvotes": 268,
            "nsfw": false,
            "greyscale": false,
            "unix_time": 1640530734,
            "width": 1816,
            "height": 2048
        },
        {
            "url": "https://i.redd.it/3eb9g3ricw781.jpg",
            "title": "Neoclassical arches // Leica M6 TTL // Kodak 400TX",
            "author": "u/photocactus",
            "permalink": "https://www.reddit.com/r/analog/comments/roxwa0/neoclassical_arches_leica_m6_ttl_kodak_400tx/",
            "upvotes": 467,
            "nsfw": false,
            "greyscale": false,
            "unix_time": 1640528683,
            "width": 2400,
            "height": 3437
        }
    ]
}
```
