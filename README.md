# AnalogDB

The collection of film photography


### Backend

* Written in [Go](https://go.dev/) with [Chi](https://github.com/go-chi/chi) HTTP router
* Consuming data from [analog-scraper](https://github.com/evanofslack/analog-scraper) 
* Images stored on [AWS S3](https://aws.amazon.com/s3/) and served from [CloudFront CDN](https://aws.amazon.com/cloudfront/)
* Metadata persisted with [Postgres](https://www.postgresql.org/)
* Deployed with [Docker](https://www.docker.com/) and [Heroku](https://www.heroku.com/)

### UI

* Powered by [Next](https://github.com/vercel/next.js/)
* With custom styles built with [CSS Modules](https://github.com/css-modules/css-modules)
* Deployed with [Vercel](https://vercel.com/).


### API

AnalogDB exposes a public REST API that provides access to thousands of film photographs.

Full documentation for the API: https://analogdb.herokuapp.com/

### Example

```bash
curl https://analogdb.herokuapp.com/latest
```

```yaml
{
   meta:{
      total_posts:1019,
      page_size:10,
      next_page_id:1640889405,
      next_page_url:/latest?page_size=10&page_id=1640889405,
   },
   posts:[
      {
       id:2170,
       images:[
         {
           resolution: low,
           url: https://d3i73ktnzbi69i.cloudfront.net/3eae28ce-2294-437d-81df-87e86cff61c3.jpeg,
           width: 216,
           height: 320,
           },
           {
           resolution: medium,
           url: https://d3i73ktnzbi69i.cloudfront.net/400abc43-b8c5-44cf-a632-c1a849b14ab4.jpeg,
           width: 519,
           height: 768,
           },
           ...
         ],
         title: The San Remo from Central Park [Leica m6, Nokton 35mm f/1.4, Portra 400],
         author: u/_35mm_,
         permalink: https://www.reddit.com/r/analog/comments/u26upj/the_san_remo_from_central_park_leica_m6_nokton/,
         upvotes: 89,
         nsfw: false,
         unix_time: 1649790635,
         sprocket: false,
      },
      ...
   ]
}
```
