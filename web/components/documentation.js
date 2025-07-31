"use client";

import { CodeHighlight } from "@mantine/code-highlight";
import { Code, Divider, Table } from "@mantine/core";
import { useBreakpoint } from "@providers/breakpoint";
import Link from "next/link";
import styles from "./documentation.module.css";
import Footer from "./footer";

export default function Documentation() {
  const breakpoints = useBreakpoint();
  let isMobile = false;
  if (breakpoints["sm"]) {
    isMobile = true;
  }

  const paginations = [
    {
      field: "page_size",
      description:
        "set the number of records to return on each page (default 20, maximum 200)",
    },
    {
      field: "page_id",
      description:
        "request a specific page of results. Each request returns a next_page_id that can be used to access the next page of results",
    },
  ];

  const paginationRows = paginations.map((page) => (
    <Table.Tr key={page.field}>
      <Table.Td>
        <Code>{page.field}</Code>
      </Table.Td>
      <Table.Td>{page.description}</Table.Td>
    </Table.Tr>
  ));

  // image resource table
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

  const imageRows = images.map((image) => (
    <Table.Tr key={image.field}>
      <Table.Td>
        <Code>{image.field}</Code>
      </Table.Td>
      <Table.Td>{image.type}</Table.Td>
      <Table.Td>{image.description}</Table.Td>
    </Table.Tr>
  ));

  // color resource table
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

  const colorRows = colors.map((color) => (
    <Table.Tr key={color.field}>
      <Table.Td>
        <Code>{color.field}</Code>
      </Table.Td>
      <Table.Td>{color.type}</Table.Td>
      <Table.Td>{color.description}</Table.Td>
    </Table.Tr>
  ));

  // keyword resource table
  const keywords = [
    { field: "word", type: "string", description: "detected keyword in image" },
    {
      field: "weight",
      type: "number",
      description: "relevance of keyword (0-1)",
    },
  ];

  const keywordRows = keywords.map((keyword) => (
    <Table.Tr key={keyword.field}>
      <Table.Td>
        <Code>{keyword.field}</Code>
      </Table.Td>
      <Table.Td>{keyword.type}</Table.Td>
      <Table.Td>{keyword.description}</Table.Td>
    </Table.Tr>
  ));

  // post resource table
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
      field: "description",
      type: "string",
      description: "post description",
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

  const postRows = posts.map((post) => (
    <Table.Tr key={post.field}>
      <Table.Td>
        <Code>{post.field}</Code>
      </Table.Td>
      <Table.Td>{post.type}</Table.Td>
      <Table.Td>{post.description}</Table.Td>
    </Table.Tr>
  ));

  // camera resource table
  const cameras = [
    { field: "id", type: "integer", description: "unique identifier" },
    {
      field: "make",
      type: "string",
      description: "camera manufacturer (e.g., nikon)",
    },
    { field: "model", type: "string", description: "camera model (e.g., fm2)" },
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

  const cameraRows = cameras.map((camera) => (
    <Table.Tr key={camera.field}>
      <Table.Td>
        <Code>{camera.field}</Code>
      </Table.Td>
      <Table.Td>{camera.type}</Table.Td>
      <Table.Td>{camera.description}</Table.Td>
    </Table.Tr>
  ));

  // film resource table
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

  const filmRows = films.map((film) => (
    <Table.Tr key={film.field}>
      <Table.Td>
        <Code>{film.field}</Code>
      </Table.Td>
      <Table.Td>{film.type}</Table.Td>
      <Table.Td>{film.description}</Table.Td>
    </Table.Tr>
  ));

  // meta resource table
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
      field: "next_page_id",
      type: "integer",
      description: "unique identifier of next page",
    },
    {
      field: "next_page_url",
      type: "string",
      description: "url path to fetch next page",
    },
    {
      field: "seed",
      type: "integer",
      description: "random seed used for random sorting",
    },
  ];

  const metaRows = metas.map((meta) => (
    <Table.Tr key={meta.field}>
      <Table.Td>
        <Code>{meta.field}</Code>
      </Table.Td>
      <Table.Td>{meta.type}</Table.Td>
      <Table.Td>{meta.description}</Table.Td>
    </Table.Tr>
  ));

  // posts general params table
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
      param: "page_id",
      description: "ID of page to retrieve",
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

  const postsGeneralRows = postsGenerals.map((general) => (
    <Table.Tr key={general.param}>
      <Table.Td>
        <Code>{general.param}</Code>
      </Table.Td>
      <Table.Td>{general.description}</Table.Td>
      <Table.Td>{general.default}</Table.Td>
      <Table.Td>{general.options}</Table.Td>
    </Table.Tr>
  ));

  // posts filter params table
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
    {
      param: "nsfw",
      description: "include nsfw (18+) images (true=only, false=exclude)",
    },
    {
      param: "grayscale",
      description:
        "include grayscale (black & white) images (true=only, false=exclude)",
    },
    {
      param: "sprocket",
      description: "include sprocket images (true=only, false=exclude)",
    },
    { param: "keyword", description: "filter by keywords (multiple allowed)" },
    { param: "color", description: "color filter (multiple allowed)" },
    {
      param: "min_color",
      description: "minimum color percentage (multiple allowed)",
    },
    { param: "width_min", description: "minimum picture width" },
    { param: "width_max", description: "maximum picture width" },
    { param: "height_min", description: "minimum picture height" },
    { param: "height_max", description: "maximum picture height" },
    { param: "ratio_min", description: "minimum picture aspect ratio" },
    { param: "ratio_max", description: "maximum picture aspect ratio" },
  ];

  const postsFilterRows = postsFilters.map((filter) => (
    <Table.Tr key={filter.param}>
      <Table.Td>
        <Code>{filter.param}</Code>
      </Table.Td>
      <Table.Td>{filter.description}</Table.Td>
    </Table.Tr>
  ));

  // cameras params table
  const camerasParams = [
    {
      param: "sort",
      description: "sort order",
      options: "alphabetical, counts",
    },
    {
      param: "page_size",
      description: "number of results to return",
      options: "integer",
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
  ];

  const camerasParamRows = camerasParams.map((param) => (
    <Table.Tr key={param.param}>
      <Table.Td>
        <Code>{param.param}</Code>
      </Table.Td>
      <Table.Td>{param.description}</Table.Td>
      <Table.Td>{param.options}</Table.Td>
    </Table.Tr>
  ));

  // films params table
  const filmsParams = [
    {
      param: "sort",
      description: "sort order",
      options: "alphabetical, counts",
    },
    {
      param: "page_size",
      description: "number of results to return",
      options: "integer",
    },
    { param: "make", description: "filter by film make", options: "string" },
    { param: "type", description: "filter by film type", options: "string" },
    { param: "speed", description: "filter by film speed", options: "integer" },
    {
      param: "colortype",
      description: "filter by color type",
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
  ];

  const filmsParamRows = filmsParams.map((param) => (
    <Table.Tr key={param.param}>
      <Table.Td>
        <Code>{param.param}</Code>
      </Table.Td>
      <Table.Td>{param.description}</Table.Td>
      <Table.Td>{param.options}</Table.Td>
    </Table.Tr>
  ));

  // similar posts params table
  const similarParams = [
    {
      param: "id",
      description: "post ID to find similar posts for (required)",
      options: "integer",
    },
    {
      param: "page_size",
      description: "maximum number of similar posts to return",
      options: "1-200 (default: 12)",
    },
    {
      param: "nsfw",
      description: "include nsfw posts in results",
      options: "boolean",
    },
    {
      param: "grayscale",
      description: "include grayscale posts in results",
      options: "boolean",
    },
    {
      param: "sprocket",
      description: "include sprocket posts in results",
      options: "boolean",
    },
  ];

  const similarParamRows = similarParams.map((param) => (
    <Table.Tr key={param.param}>
      <Table.Td>
        <Code>{param.param}</Code>
      </Table.Td>
      <Table.Td>{param.description}</Table.Td>
      <Table.Td>{param.options}</Table.Td>
    </Table.Tr>
  ));

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
            change. All film pictures are scrapped from{" "}
            <u>
              <Link href="https://www.reddit.com/r/analog/">reddit</Link>
            </u>
            . All credit goes to the original photographers.
          </p>
          <p>
            Use the following URI to access the endpoints:{" "}
            <Code>https://api.analogdb.com/v1</Code>
          </p>
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
          <h1 className={styles.h1}> Rate Limiting </h1>
          <p>
            The Analogdb API currently places a limit of 60 requests/min.
            Current rate limit status is returned in response headers after each
            request including remaining requests and reset time in unix epoch
            seconds.
          </p>
          <Code block>
            x-ratelimit-limit: 60
            <br></br>x-ratelimit-remaining: 59
            <br></br>x-ratelimit-reset: 1691712960
          </Code>
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
          <h1 className={styles.h1}> Pagination </h1>
          <p>
            All collection endpoints are paginated with keyset pagination. By
            default, 20 records are returned per page. Pagination can be
            controlled with the following parameters:
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>param</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{paginationRows}</Table.Tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <CodeHighlight
                code="curl https://api.analogdb.com/posts?page_size=10&page_id=774"
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
          )}
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
          <h1 className={styles.h1}> Resources </h1>
          <h2 className={styles.h2}> Image </h2>
          <p>
            The <Code>image</Code> resource contains the image URL as well as
            resolution and dimensions.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>field name</Table.Th>
                <Table.Th>type</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{imageRows}</Table.Tbody>
          </Table>
          <h2 className={styles.h2}> Color </h2>
          <p>
            The <Code>color</Code> resource represents primary colors extracted
            from images and corresponding percentages.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>field name</Table.Th>
                <Table.Th>type</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{colorRows}</Table.Tbody>
          </Table>
          <h2 className={styles.h2}> Keyword </h2>
          <p>
            The <Code>keyword</Code> resource contains keywords scraped from
            post along with relevance score.
          </p>
          <Table highlightOnHover withColumnBorders withRowBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>field name</Table.Th>
                <Table.Th>type</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{keywordRows}</Table.Tbody>
          </Table>
          <h2 className={styles.h2}> Post </h2>
          <p>
            The <Code>post</Code> resource contains a list of <Code>image</Code>
            (multiple resolutions) as well as metadata about the post including
            timestamp, score, camera, film, colors, keywords, etc.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>field name</Table.Th>
                <Table.Th>type</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{postRows}</Table.Tbody>
          </Table>
          <h2 className={styles.h2}> Camera </h2>
          <p>
            The <Code>camera</Code> resource contains film cameras used in posts
            including manufacturer, model, and description and post count.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>field name</Table.Th>
                <Table.Th>type</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{cameraRows}</Table.Tbody>
          </Table>
          <h2 className={styles.h2}> Film </h2>
          <p>
            The <Code>film</Code> resource contains film stocks used in posts,
            including manufacturer, type, speed, and post count.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>field name</Table.Th>
                <Table.Th>type</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{filmRows}</Table.Tbody>
          </Table>
          <h2 className={styles.h2}> Meta </h2>
          <p>
            The <Code>meta</Code> resource contains supplementary information
            for a collection of <Code>post</Code> resources, including
            pagination details and total counts.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>field name</Table.Th>
                <Table.Th>type</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{metaRows}</Table.Tbody>
          </Table>
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
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
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>param</Table.Th>
                <Table.Th>description</Table.Th>
                <Table.Th>default</Table.Th>
                <Table.Th>options</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{postsGeneralRows}</Table.Tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <CodeHighlight
                code="curl https://api.analogdb.com/posts?sort=score&page_size=50"
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
          )}
          <h3 className={styles.h3}>Filter Parameters</h3>
          <p>
            Posts can be filtered by various criteria including camera, film,
            time, colors, keywords, and image dimensions. For boolean filters
            (nsfw, grayscale, sprocket) if not provided, all posts are included;
            if set to <Code>true</Code>, only that type is returned; if set to{" "}
            <Code>false</Code>, that type is excluded.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>param</Table.Th>
                <Table.Th>description</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{postsFilterRows}</Table.Tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <CodeHighlight
                code="curl 'https://api.analogdb.com/posts?camera_make=nikon&film_make=kodak&grayscale=false&keyword=portrait'"
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
          )}
          <h2 className={styles.h2}> /post/:id </h2>
          <p>
            Returns a single specific <Code>post</Code> resource as identified
            by ID.
          </p>
          {!isMobile && (
            <div className={styles.codeblock}>
              <CodeHighlight
                code="curl https://api.analogdb.com/post/1924"
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
          )}
          <h2 className={styles.h2}> /post/:id/similar </h2>
          <p>
            Returns a collection of <Code>post</Code> resources that are
            visually similar to the specified post based on vector similarity of
            image embeddings.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>param</Table.Th>
                <Table.Th>description</Table.Th>
                <Table.Th>options</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{similarParamRows}</Table.Tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <CodeHighlight
                code="curl https://api.analogdb.com/post/1924/similar?page_size=20&nsfw=false"
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
          )}
          <h2 className={styles.h2}> /cameras </h2>
          <p>
            Returns a collection of <Code>camera</Code> resources with optional
            filtering and sorting. Useful for discovering cameras (not
            extensive) and their post counts.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>param</Table.Th>
                <Table.Th>description</Table.Th>
                <Table.Th>options</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{camerasParamRows}</Table.Tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <CodeHighlight
                code="curl https://api.analogdb.com/cameras?sort=counts&make=nikon&include_counts=true"
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
          )}
          <h2 className={styles.h2}> /films </h2>
          <p>
            Returns a collection of <Code>film</Code> resources with optional
            filtering and sorting. Useful for discovering film stocks (not
            extensive) and their post counts.
          </p>
          <Table highlightOnHover withColumnBorders>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>param</Table.Th>
                <Table.Th>description</Table.Th>
                <Table.Th>options</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>{filmsParamRows}</Table.Tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <CodeHighlight
                code="curl https://api.analogdb.com/films?sort=counts&make=kodak&speed=400&include_zero_counts=true"
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
          )}
        </div>
      </div>
      <Footer />
    </main>
  );
}
