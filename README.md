# AnalogDB

The collection of film photography


### About

[AnalogDB](https://analogdb.com) provides a large collection of curated analog photographs to users through a REST API interface. Beyond just returning photos, AnalogDB enables discovery of similar images, provides relevant keywords, extracts dominant colors, and allows for filtering, sorting and searching across all images. 

### Design

AnalogDB makes use of several technologies and services to enable a full featured product. 

<img alt="analogdb-diagram" src="https://github.com/evanofslack/analogdb/assets/51209817/cd0f5de5-32be-44af-914e-4cdadb8b2bdf">
<br/><br/>

Data is scraped from reddit and ingested with [analogdb-scraper](https://github.com/evanofslack/analogdb-scraper), a python service. In addition to scraping, this service is responsible for transforming raw images, extraction of keywords and colors, uploading to [AWS S3](https://aws.amazon.com/s3/), and creation of resources through the backend api. Images from S3 are served from [CloudFront CDN](https://aws.amazon.com/cloudfront/) for quick and reliable delievery. 

The core backend application is written in [Go](https://go.dev/) and makes use of [Chi](https://github.com/go-chi/chi) as an HTTP router. It exposes handlers that are responsible for parsing authentication headers, filtering incoming requests, querying databases, and returning JSON responses. Upon upload, all images are transformed with the [ResNet-50 CNN](https://datagen.tech/guides/computer-vision/resnet-50/) to create embeddings which are stored in a [Weaviate](https://github.com/weaviate/weaviate) vector database. The backend is packaged as several docker containers and hosted on a VPS.

The frontend web application is built with [Next.js](https://github.com/vercel/next.js/), making use of server-side rendering and incremental static regeneration for quick loading pages. [Zustand](https://github.com/pmndrs/zustand) is utilized for state management. All styles are built from scratch with [CSS Modules](https://github.com/css-modules/css-modules). The frontend is currently deployed with [Vercel](https://vercel.com/). 


### API

Full documentation for the API: https://api.analogdb.com/

### Example

```bash
curl https://api.analogdb.com/posts
```

```yaml
{
   meta:{
      total_posts:5842,
      page_size:20,
      next_page_id:1684684780,
      next_page_url:"/posts?sort=latest&page_size=20&page_id=1684684780",
   },
   posts: [
      {
       id:7378,
       title: A Forest on the Coast | Portra 400 | Canon 1V | 50mm,
       author: navazuals,
       permalink: https://www.reddit.com/r/analog/comments/13p9lme/a_forest_on_the_coast_portra_400_canon_1v_50mm/,
       upvotes: 89,
       unix_time: 1684804283,
       nsfw: false,
       sprocket: false,
       images: [
       {
         resolution: low,
         url: https://d3i73ktnzbi69i.cloudfront.net/505e03d0-e6c2-4596-97d2-77d6831e802c.jpeg,
         width: 477,
         weight: 720,
       },
       {
         resolution: medium,
         url: https://d3i73ktnzbi69i.cloudfront.net/0149e7c5-cefe-4cfa-a731-c7696c067d98.jpeg,
         width: 716,
         height: 1080,
       },
       ...
       ],
       colors: [
       {
         hex: #5d5933
         css: darkolivegreen
         percent: 0.33687689
       },
       {
         hex: #c6c5b1
         css: silver
         percent: 0.24639529
       },
       ],
       ...
       keywords: [
       {
         word: portra
         weight: 0.2
       },
       {
         word: forest
         weight: 0.2
       },
       ...
      ],
    },
    ...
  ]
}
```

### Deploying

There are prebuilt docker images at `evanofslack/analogdb:latest`

Please see [docker-compose.yaml](https://github.com/evanofslack/analogdb/blob/main/docker-compose.yml) for an example compose deployment with necessary variables and services.

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
