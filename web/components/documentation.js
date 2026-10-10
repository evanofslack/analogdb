"use client";

import { CodeHighlight } from "@mantine/code-highlight";
import { Code, Divider, Table } from "@mantine/core";
import Link from "next/link";
import styles from "./documentation.module.css";
import Footer from "./footer";

// field name, type, description
function FieldTable({ rows }) {
  return (
    <Table.ScrollContainer minWidth={0} type="native">
      <Table highlightOnHover withColumnBorders>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>field name</Table.Th>
            <Table.Th>type</Table.Th>
            <Table.Th>description</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {rows.map((row) => (
            <Table.Tr key={row.field}>
              <Table.Td>
                <Code>{row.field}</Code>
              </Table.Td>
              <Table.Td>{row.type}</Table.Td>
              <Table.Td>{row.description}</Table.Td>
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Table.ScrollContainer>
  );
}

// param and description, plus any extra columns
function ParamTable({ rows, columns = [] }) {
  return (
    <Table.ScrollContainer minWidth={0} type="native">
      <Table highlightOnHover withColumnBorders>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>param</Table.Th>
            <Table.Th>description</Table.Th>
            {columns.map((column) => (
              <Table.Th key={column}>{column}</Table.Th>
            ))}
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {rows.map((row) => (
            <Table.Tr key={row.param}>
              <Table.Td>
                <Code>{row.param}</Code>
              </Table.Td>
              <Table.Td>{row.description}</Table.Td>
              {columns.map((column) => (
                <Table.Td key={column}>{row[column]}</Table.Td>
              ))}
            </Table.Tr>
          ))}
        </Table.Tbody>
      </Table>
    </Table.ScrollContainer>
  );
}

function Example({ code }) {
  return (
    <div className={styles.codeblock}>
      <CodeHighlight
        code={code}
        language="bash"
        copyLabel="copy example"
        copiedLabel="copied"
        styles={{
          code: {
            fontSize: "0.75rem",
          },
        }}
      />
    </div>
  );
}

function Section() {
  return (
    <div className={styles.divider}>
      <Divider my="sm" />
    </div>
  );
}

const flagDescription = (name) =>
  `filter ${name} posts (true=only, false=exclude)`;

const flagParams = [
  {
    param: "nsfw",
    description: flagDescription("nsfw (18+)"),
    options: "boolean",
  },
  {
    param: "grayscale",
    description: flagDescription("grayscale (black & white)"),
    options: "boolean",
  },
  {
    param: "sprocket",
    description: flagDescription("sprocket"),
    options: "boolean",
  },
];

const paginations = [
  {
    param: "page_size",
    description:
      "set the number of records to return on each page (default 20, maximum 200)",
  },
  {
    param: "cursor",
    description:
      "request the next page of results. Each request returns a next_cursor that can be passed back to access the next page of results",
  },
  {
    param: "page_id",
    description: "deprecated, use cursor",
  },
];

const errorStatuses = [
  { param: "400", description: "invalid parameter or request" },
  { param: "404", description: "post or keyword not found" },
  { param: "413", description: "uploaded image is larger than 5 MB" },
  { param: "415", description: "uploaded image is not JPEG, PNG or WebP" },
  { param: "429", description: "rate limit exceeded" },
  { param: "500", description: "internal server error" },
  { param: "503", description: "search is busy, try again shortly" },
];

const images = [
  { field: "url", type: "string", description: "link to image" },
  {
    field: "resolution",
    type: "string",
    description: "low, medium, high, raw",
  },
  {
    field: "width",
    type: "integer",
    description: "width of image in pixels",
  },
  {
    field: "height",
    type: "integer",
    description: "height of image in pixels",
  },
];

const colors = [
  {
    field: "css",
    type: "string",
    description: "CSS color name (e.g., dimgray)",
  },
  {
    field: "hex",
    type: "string",
    description: "hex color code (e.g., #837d5c)",
  },
  {
    field: "html",
    type: "string",
    description: "HTML color name (e.g., gray)",
  },
  {
    field: "percent",
    type: "number",
    description: "percent of image this color represents",
  },
];

const keywords = [
  { field: "word", type: "string", description: "detected keyword in image" },
  {
    field: "weight",
    type: "number",
    description: "relevance of keyword (0-1)",
  },
];

const posts = [
  { field: "id", type: "integer", description: "unique identifier" },
  { field: "title", type: "string", description: "title of post" },
  { field: "author", type: "string", description: "author of post" },
  { field: "permalink", type: "string", description: "url of post source" },
  { field: "score", type: "integer", description: "total votes of post" },
  {
    field: "timestamp",
    type: "integer",
    description: "time of post creation (unix time)",
  },
  {
    field: "created",
    type: "string",
    description: "time the post was added to AnalogDB (RFC 3339)",
  },
  {
    field: "updated",
    type: "string",
    description: "time the post was last changed in AnalogDB (RFC 3339)",
  },
  {
    field: "description",
    type: "string",
    description: "post description",
  },
  {
    field: "caption",
    type: "string",
    description: "generated description of the image",
  },
  {
    field: "camera_make",
    type: "string",
    description: "camera manufacturer (e.g., nikon)",
  },
  {
    field: "camera_model",
    type: "string",
    description: "camera model (e.g., fm2)",
  },
  {
    field: "film_make",
    type: "string",
    description: "film manufacturer (e.g., kodak)",
  },
  {
    field: "film_type",
    type: "string",
    description: "film type (e.g., portra 400)",
  },
  {
    field: "film_speed",
    type: "integer",
    description: "film ISO speed (e.g., 400)",
  },
  {
    field: "aperture",
    type: "string",
    description: "aperture setting (e.g., f/2.0)",
  },
  {
    field: "focal_length",
    type: "integer",
    description: "focal length in mm (e.g., 35)",
  },
  {
    field: "nsfw",
    type: "bool",
    description: "image is NSFW (not safe for work, 18+)",
  },
  {
    field: "grayscale",
    type: "bool",
    description: "image is grayscale (black & white)",
  },
  {
    field: "sprocket",
    type: "bool",
    description: "image is a sprocket shot (exposed film sprockets)",
  },
  {
    field: "images",
    type: "array[image]",
    description: "list of images at different resolutions",
  },
  {
    field: "colors",
    type: "array[color]",
    description: "dominant colors extracted from image",
  },
  {
    field: "keywords",
    type: "array[keyword]",
    description: "keywords extracted from post",
  },
];

const catalogPosts = [
  { field: "id", type: "integer", description: "unique identifier" },
  { field: "title", type: "string", description: "title of post" },
  { field: "score", type: "integer", description: "total votes of post" },
  {
    field: "images",
    type: "array[image]",
    description: "list of images at different resolutions",
  },
];

const cameras = [
  { field: "id", type: "integer", description: "unique identifier" },
  {
    field: "make",
    type: "string",
    description: "camera manufacturer (e.g., nikon)",
  },
  { field: "model", type: "string", description: "camera model (e.g., fm2)" },
  { field: "slug", type: "string", description: "url name (e.g., nikon-fm2)" },
  {
    field: "description",
    type: "string",
    description: "description of camera",
  },
  {
    field: "post_count",
    type: "integer",
    description: "number of posts using this camera",
  },
  {
    field: "top_posts",
    type: "array[catalog post]",
    description: "highest scoring posts, only when top_posts is set",
  },
  {
    field: "created",
    type: "string",
    description: "timestamp when camera was added",
  },
  {
    field: "updated",
    type: "string",
    description: "timestamp when camera was last updated",
  },
];

const films = [
  { field: "id", type: "integer", description: "unique identifier" },
  {
    field: "make",
    type: "string",
    description: "film manufacturer (e.g., kodak)",
  },
  {
    field: "type",
    type: "string",
    description: "film type (e.g., portra 400)",
  },
  {
    field: "slug",
    type: "string",
    description: "url name (e.g., kodak-portra-400)",
  },
  {
    field: "speed",
    type: "integer",
    description: "film ISO speed (e.g., 400)",
  },
  {
    field: "color_type",
    type: "string",
    description: "color or black & white film",
  },
  {
    field: "description",
    type: "string",
    description: "description of film",
  },
  {
    field: "post_count",
    type: "integer",
    description: "number of posts using this film",
  },
  {
    field: "top_posts",
    type: "array[catalog post]",
    description: "highest scoring posts, only when top_posts is set",
  },
  {
    field: "created",
    type: "string",
    description: "timestamp when film was added",
  },
  {
    field: "updated",
    type: "string",
    description: "timestamp when film was last updated",
  },
];

const metas = [
  {
    field: "total_posts",
    type: "integer",
    description: "total number of posts served by endpoint query",
  },
  {
    field: "page_size",
    type: "integer",
    description: "maximum number of posts returned per page",
  },
  {
    field: "next_cursor",
    type: "string",
    description: "cursor for the next page, empty on the last page",
  },
  {
    field: "next_page_id",
    type: "integer",
    description: "deprecated, use next_cursor",
  },
  {
    field: "next_page_url",
    type: "string",
    description: "path of the next page, relative to /v1",
  },
  {
    field: "seed",
    type: "integer",
    description: "random seed used for random sorting",
  },
];

const searchResults = [
  {
    field: "meta",
    type: "object",
    description: "page_size, next_cursor and next_page_url, as in meta",
  },
  { field: "posts", type: "array[post]", description: "matching posts" },
  {
    field: "related_keywords",
    type: "array[string]",
    description: "keywords related to the search, first page only",
  },
];

const postsGenerals = [
  {
    param: "sort",
    description: "how to order the posts",
    default: "time",
    options: "time, score, random",
  },
  {
    param: "page_size",
    description: "maximum number of posts returned",
    default: "20",
    options: "1-200",
  },
  {
    param: "cursor",
    description: "cursor of page to retrieve, from next_cursor",
    default: "null",
    options: "",
  },
  {
    param: "page_id",
    description: "deprecated, use cursor",
    default: "null",
    options: "",
  },
  {
    param: "seed",
    description: "random seed for consistent random sorting",
    default: "null",
    options: "integer",
  },
];

const postsFilters = [
  { param: "id", description: "filter by post ID" },
  { param: "title", description: "filter by post title" },
  { param: "author", description: "filter by author" },
  {
    param: "time_start",
    description: "filter by start time (unix timestamp)",
  },
  { param: "time_end", description: "filter by end time (unix timestamp)" },
  { param: "camera_make", description: "filter by camera make" },
  { param: "camera_model", description: "filter by camera model" },
  { param: "film_make", description: "filter by film make" },
  { param: "film_type", description: "filter by film type" },
  { param: "film_speed", description: "filter by film speed" },
  { param: "focal_length", description: "filter by focal length" },
  { param: "aperture", description: "filter by aperture" },
  ...flagParams,
  {
    param: "keyword",
    description:
      "filter by keyword, repeat to require every keyword (e.g., keyword=beach&keyword=sunset)",
  },
  {
    param: "color",
    description:
      "filter by HTML color name (e.g., gray), repeat for more colors",
  },
  {
    param: "min_color",
    description:
      "minimum percent of the image for each color, paired by position with color",
  },
  { param: "width_min", description: "minimum picture width" },
  { param: "width_max", description: "maximum picture width" },
  { param: "height_min", description: "minimum picture height" },
  { param: "height_max", description: "maximum picture height" },
  { param: "ratio_min", description: "minimum picture aspect ratio" },
  { param: "ratio_max", description: "maximum picture aspect ratio" },
];

const catalogParams = [
  {
    param: "min_count",
    description:
      "only return entries with at least this many posts, implies include_counts",
    options: "integer",
  },
  {
    param: "top_posts",
    description: "attach this many top scoring posts to each entry",
    options: "1-10",
  },
];

const camerasParams = [
  {
    param: "sort",
    description: "sort order",
    options: "alphabetical, counts",
  },
  {
    param: "page_size",
    description: "number of results to return",
    options: "1-1000",
  },
  { param: "make", description: "filter by camera make", options: "string" },
  {
    param: "model",
    description: "filter by camera model",
    options: "string",
  },
  {
    param: "id",
    description: "filter by specific camera ID",
    options: "integer",
  },
  {
    param: "include_counts",
    description: "include post counts",
    options: "boolean",
  },
  {
    param: "exclude_zero_counts",
    description: "exclude cameras with zero post counts",
    options: "boolean",
  },
  ...catalogParams,
];

const filmsParams = [
  {
    param: "sort",
    description: "sort order",
    options: "alphabetical, counts",
  },
  {
    param: "page_size",
    description: "number of results to return",
    options: "1-1000",
  },
  { param: "make", description: "filter by film make", options: "string" },
  { param: "type", description: "filter by film type", options: "string" },
  { param: "speed", description: "filter by film speed", options: "integer" },
  {
    param: "colortype",
    description: "filter by color type (the color_type field)",
    options: "string",
  },
  {
    param: "id",
    description: "filter by specific film ID",
    options: "integer",
  },
  {
    param: "include_counts",
    description: "include post counts",
    options: "boolean",
  },
  {
    param: "exclude_zero_counts",
    description: "exclude films with zero post counts",
    options: "boolean",
  },
  ...catalogParams,
];

const similarParams = [
  {
    param: "page_size",
    description: "maximum number of similar posts to return",
    options: "1-50 (default: 12)",
  },
  ...flagParams,
];

const searchParams = [
  {
    param: "q",
    description: "search text (required)",
    options: "1-200 characters",
  },
  {
    param: "page_size",
    description: "number of posts per page",
    options: "1-100 (default: 50)",
  },
  {
    param: "cursor",
    description: "cursor of page to retrieve, from next_cursor",
    options: "string",
  },
  ...flagParams,
];

const imageSearchParams = [
  {
    param: "image",
    description: "image to search with, as multipart form data (required)",
    options: "JPEG, PNG, WebP up to 5 MB",
  },
  {
    param: "page_size",
    description: "number of posts to return",
    options: "1-100 (default: 50)",
  },
  ...flagParams,
];

const keywordParams = [
  {
    param: "top_posts",
    description: "number of top scoring posts",
    options: "1-10 (default: 6)",
  },
  {
    param: "related",
    description: "number of related keywords",
    options: "0-30 (default: 12)",
  },
];

const keywordsSummaryParams = [
  {
    param: "page_size",
    description: "number of keywords to return",
    options: "1-500 (default: 50)",
  },
  {
    param: "days",
    description: "only count posts created in the last days",
    options: "1-90",
  },
  {
    param: "min_count",
    description: "only return keywords on at least this many posts",
    options: "integer",
  },
  {
    param: "top_posts",
    description: "attach this many top scoring posts to each keyword",
    options: "1-10",
  },
];

export default function Documentation() {
  return (
    <main>
      <div className={styles.center}>
        <div className={styles.container}>
          <h1 className={styles.h1}> Overview </h1>
          <p>
            This document outlines the AnalogDB API. This API provides film
            photographs and metadata in JSON form as a REST-style service. The
            API is open-source and available on{" "}
            <u>
              <Link href="https://github.com/evanofslack/analogdb">github</Link>
            </u>
            . The swagger docs are available{" "}
            <u>
              <Link href="https://api.analogdb.com/swagger/index.html">
                here
              </Link>
            </u>
            .
          </p>
          <p>
            The AnalogDB project is currently under development and subject to
            change. All film pictures are scraped from{" "}
            <u>
              <Link href="https://www.reddit.com/r/analog/">reddit</Link>
            </u>
            . All credit goes to the original photographers.
          </p>
          <p>
            Use the following URI to access the endpoints:{" "}
            <Code>https://api.analogdb.com/v1</Code>. Paths without{" "}
            <Code>/v1</Code> still work but are deprecated and will stop working
            on March 31, 2027.
          </p>
          <Section />
          <h1 className={styles.h1}> Rate Limiting </h1>
          <p>
            The Analogdb API currently places a limit of 60 requests/min.
            Current rate limit status is returned in response headers after each
            request including remaining requests and reset time in unix epoch
            seconds. Requests over the limit return status <Code>429</Code>.
          </p>
          <Code block>
            x-ratelimit-limit: 60
            <br></br>x-ratelimit-remaining: 59
            <br></br>x-ratelimit-reset: 1691712960
          </Code>
          <Section />
          <h1 className={styles.h1}> Errors </h1>
          <p>
            Errors return a non 2xx status with a JSON body holding a message.
          </p>
          <Code block>{'{"error": "post not found"}'}</Code>
          <ParamTable rows={errorStatuses} />
          <Section />
          <h1 className={styles.h1}> Pagination </h1>
          <p>
            <Code>/posts</Code> and <Code>/search</Code> are paginated with
            cursors. By default, 20 posts are returned per page. Pass the{" "}
            <Code>next_cursor</Code> from one response to get the next page, or
            follow <Code>next_page_url</Code>, a path relative to{" "}
            <Code>/v1</Code>. Other endpoints return a single page sized by{" "}
            <Code>page_size</Code>.
          </p>
          <ParamTable rows={paginations} />
          <Example code="curl https://api.analogdb.com/v1/posts?page_size=10&cursor=eyJzIjoidGltZSIsInYiOjE3OTA1Mzk5MjIsImlkIjo0MDA1Mn0" />
          <Section />
          <h1 className={styles.h1}> Resources </h1>
          <h2 className={styles.h2}> Image </h2>
          <p>
            The <Code>image</Code> resource contains the image URL as well as
            resolution and dimensions.
          </p>
          <FieldTable rows={images} />
          <h2 className={styles.h2}> Color </h2>
          <p>
            The <Code>color</Code> resource represents primary colors extracted
            from images and corresponding percentages.
          </p>
          <FieldTable rows={colors} />
          <h2 className={styles.h2}> Keyword </h2>
          <p>
            The <Code>keyword</Code> resource contains keywords scraped from
            post along with relevance score.
          </p>
          <FieldTable rows={keywords} />
          <h2 className={styles.h2}> Post </h2>
          <p>
            The <Code>post</Code> resource contains a list of <Code>image</Code>
            (multiple resolutions) as well as metadata about the post including
            timestamp, score, camera, film, colors, keywords, etc. Fields that
            are unknown, such as camera or film, are left out.
          </p>
          <FieldTable rows={posts} />
          <h2 className={styles.h2}> Catalog Post </h2>
          <p>
            The <Code>catalog post</Code> resource is a short form of a{" "}
            <Code>post</Code>, attached to cameras, films and keywords.
          </p>
          <FieldTable rows={catalogPosts} />
          <h2 className={styles.h2}> Camera </h2>
          <p>
            The <Code>camera</Code> resource contains film cameras used in posts
            including manufacturer, model, and description and post count.
          </p>
          <FieldTable rows={cameras} />
          <h2 className={styles.h2}> Film </h2>
          <p>
            The <Code>film</Code> resource contains film stocks used in posts,
            including manufacturer, type, speed, and post count.
          </p>
          <FieldTable rows={films} />
          <h2 className={styles.h2}> Meta </h2>
          <p>
            The <Code>meta</Code> resource contains supplementary information
            for a collection of <Code>post</Code> resources, including
            pagination details and total counts.
          </p>
          <FieldTable rows={metas} />
          <h2 className={styles.h2}> Search Result </h2>
          <p>
            The <Code>search result</Code> resource is returned by both search
            endpoints.
          </p>
          <FieldTable rows={searchResults} />
          <Section />
          <h1 className={styles.h1}> Endpoints </h1>
          <h2 className={styles.h2}> /posts </h2>
          <p>
            Returns a collection of <Code>post</Code> resources with
            accompanying <Code>meta</Code> resource. Supports extensive
            filtering and sorting options.
          </p>
          <h3 className={styles.h3}>General Parameters</h3>
          <p>
            Posts can be sorted by time, score, or pseudo-randomly. Limits can
            be placed for maximum number of returned posts. If total number of
            posts exceeds the limit, results will be paginated.
          </p>
          <ParamTable rows={postsGenerals} columns={["default", "options"]} />
          <Example code="curl https://api.analogdb.com/v1/posts?sort=score&page_size=50" />
          <h3 className={styles.h3}>Filter Parameters</h3>
          <p>
            Posts can be filtered by various criteria including camera, film,
            time, colors, keywords, and image dimensions. For boolean filters
            (nsfw, grayscale, sprocket) if not provided, all posts are included;
            if set to <Code>true</Code>, only that type is returned; if set to{" "}
            <Code>false</Code>, that type is excluded.
          </p>
          <ParamTable rows={postsFilters} />
          <Example code="curl 'https://api.analogdb.com/v1/posts?camera_make=nikon&film_make=kodak&grayscale=false&keyword=portrait'" />
          <h2 className={styles.h2}> /post/:id </h2>
          <p>
            Returns a single specific <Code>post</Code> resource as identified
            by ID.
          </p>
          <Example code="curl https://api.analogdb.com/v1/post/1924" />
          <h2 className={styles.h2}> /post/:id/similar </h2>
          <p>
            Returns a collection of <Code>post</Code> resources that are
            visually similar to the specified post based on vector similarity of
            image embeddings. Results are a single page.
          </p>
          <ParamTable rows={similarParams} columns={["options"]} />
          <Example code="curl 'https://api.analogdb.com/v1/post/1924/similar?page_size=20&nsfw=false'" />
          <h2 className={styles.h2}> /search </h2>
          <p>
            Returns a <Code>search result</Code> of posts matching a text query,
            searching image content, keywords, captions and titles together.
            Paginated with <Code>cursor</Code>.
          </p>
          <ParamTable rows={searchParams} columns={["options"]} />
          <Example code="curl 'https://api.analogdb.com/v1/search?q=girl+at+sunset&grayscale=false'" />
          <h2 className={styles.h2}> /search/image </h2>
          <p>
            A <Code>POST</Code> that returns a <Code>search result</Code> of
            posts that look like an uploaded image. Results are a single page.
          </p>
          <ParamTable rows={imageSearchParams} columns={["options"]} />
          <Example code="curl -F image=@photo.jpg 'https://api.analogdb.com/v1/search/image?page_size=20'" />
          <h2 className={styles.h2}> /cameras </h2>
          <p>
            Returns a collection of <Code>camera</Code> resources with optional
            filtering and sorting. Useful for discovering cameras (not
            extensive) and their post counts.
          </p>
          <ParamTable rows={camerasParams} columns={["options"]} />
          <Example code="curl 'https://api.analogdb.com/v1/cameras?sort=counts&make=nikon&include_counts=true'" />
          <h2 className={styles.h2}> /films </h2>
          <p>
            Returns a collection of <Code>film</Code> resources with optional
            filtering and sorting. Useful for discovering film stocks (not
            extensive) and their post counts.
          </p>
          <ParamTable rows={filmsParams} columns={["options"]} />
          <Example code="curl 'https://api.analogdb.com/v1/films?sort=counts&make=kodak&speed=400&exclude_zero_counts=true'" />
          <h2 className={styles.h2}> /keyword/:word </h2>
          <p>
            Returns the post <Code>count</Code> of one keyword, its{" "}
            <Code>top_posts</Code> as <Code>catalog post</Code> resources and{" "}
            <Code>related</Code> keywords that often appear with it. URL encode
            the word.
          </p>
          <ParamTable rows={keywordParams} columns={["options"]} />
          <Example code="curl https://api.analogdb.com/v1/keyword/beach?related=6" />
          <h2 className={styles.h2}> /keywords/summary </h2>
          <p>
            Returns the most common <Code>keywords</Code>, each with a{" "}
            <Code>word</Code> and post <Code>count</Code>.
          </p>
          <ParamTable rows={keywordsSummaryParams} columns={["options"]} />
          <Example code="curl 'https://api.analogdb.com/v1/keywords/summary?page_size=20&days=30'" />
        </div>
      </div>
      <Footer />
    </main>
  );
}
