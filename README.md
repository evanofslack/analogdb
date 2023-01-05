# AnalogDB

The collection of film photography


### About

AnalogDB provides a large collection of curated analog photographs to users through a REST API interface. Beyond just returning photos, AnalogDB provides methods for sorting by time or popularity, enables filtering by nsfw, black & white, and exposed sprockets, and allows for querying by film stock, camera model and camera settings. 

### Design

AnalogDB makes use of several technologies and services to enable a full featured product. 

<img width="681" alt="Screen Shot 2022-07-19 at 8 50 39 PM" src="https://user-images.githubusercontent.com/51209817/179872652-32c019e3-2e3c-4086-84fe-b6e149522e2d.png">

Data is scraped from reddit and ingested with [analogdb-scraper](https://github.com/evanofslack/analogdb-scraper), a python service. In addition to scraping, this service is responsible for transforming raw images, uploading to [AWS S3](https://aws.amazon.com/s3/), and creating resources through the backend api. Images from S3 are served from [CloudFront CDN](https://aws.amazon.com/cloudfront/) for quick and reliable delievery. 

The core backend application is written in [Go](https://go.dev/) and makes use of [Chi](https://github.com/go-chi/chi) as an HTTP router. It exposes handlers that are responsible for filtering incoming requests, establishing connections with Postgres, and returning JSON responses. The Go application and Postgres database are containerized with [Docker](https://www.docker.com/) for reliable development and deployment. The backend is currently hosted on on a private VPS.

The frontend web application is built with [Next.js](https://github.com/vercel/next.js/), making use of server-side rendering and incremental static regeneration for quick loading pages. [Zustand](https://github.com/pmndrs/zustand) is utilized for state management. All styles are built from scratch with [CSS Modules](https://github.com/css-modules/css-modules). The frontend is currently deployed with [Vercel](https://vercel.com/). 


### API

Full documentation for the API: https://api.analogdb.com/

### Example

```bash
curl https://api.analogdb.com/posts/latest
```

```yaml
{
   meta:{
      total_posts:3306,
      page_size:20,
      next_page_id:1640889405,
      next_page_url:/posts/latest?page_size=20&page_id=1640889405,
   },
   posts:[
      {
       id:2170,
       title: The San Remo from Central Park [Leica m6, Nokton 35mm f/1.4, Portra 400],
       author: u/_35mm_,
       permalink: https://www.reddit.com/r/analog/comments/u26upj/the_san_remo_from_central_park_leica_m6_nokton/,
       upvotes: 89,
       nsfw: false,
       unix_time: 1649790635,
       sprocket: false,
       images:[
       {
         resolution: low,
         url: https://d3i73ktnzbi69i.cloudfront.net/3eae28ce-2294-437d-81df-87e86cff61c3.jpeg,
         width: 216,
         weight: 320,
       },
       {
         resolution: medium,
         url: https://d3i73ktnzbi69i.cloudfront.net/400abc43-b8c5-44cf-a632-c1a849b14ab4.jpeg,
         width: 519,
         height: 768,
       },
       ...
      ],
    },
    ...
  ]
}
```

### Developing

Docker and docker-compose can be utilized for a consistent development experience. 

To spin up the backend and database:

`docker-compose -f docker-compose-dev.yaml up`

To run backend unit tests:

`go test ./...`

To serve the frontend locally:

`cd web && npm run dev`

### Contributing

All contributions are welcomed and encouraged. Please create a new issue to discuss potential improvements or submit a pull request. 
