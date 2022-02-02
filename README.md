# AnalogDB API

API serving film photographs scraped from [r/analog](https://www.reddit.com/r/analog/)

* Built with [Go](https://go.dev/), [Chi](https://github.com/go-chi/chi), and [Postgres](https://www.postgresql.org/)
* Deployed with [Docker](https://www.docker.com/) and [Heroku](https://www.heroku.com/)

### Demo

See the API in action: https://www.analogdb.com/

### Example

Full documentation for the API: https://analogdb.herokuapp.com/

```bash
curl https://analogdb.herokuapp.com/latest
```

```yaml
{
   "meta":{
      "total_posts":35,
      "page_size":10,
      "next_page_id":"1640889405",
      "next_page_url":"/latest?page_size=10&page_id=1640889405"
   },
   "posts":[
      {
         "id":110,
         "url":"https://d3i73ktnzbi69i.cloudfront.net/110.jpeg",
         "title":"2021 thru my eyes | leica m6 | summicron 35mm | various",
         "author":"u/basedjason",
         "permalink":"https://www.reddit.com/r/analog/comments/rto4fq/2021_thru_my_eyes_leica_m6_summicron_35mm_various/",
         "upvotes":38,
         "nsfw":false,
         "grayscale":false,
         "unix_time":1641058655,
         "width":2051,
         "height":2564
         "sprocket": false
      },
      ...
   ]
}
```
