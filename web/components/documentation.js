import styles from "./documentation.module.css";
import Footer from "./footer";
import Link from "next/link";
import { Prism } from "@mantine/prism";
import { Table, Code, Divider } from "@mantine/core";
import { useBreakpoint } from "../providers/breakpoint.js";

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
    <tr key={page.field}>
      <td>
        <Code>{page.field}</Code>
      </td>
      <td>{page.description}</td>
    </tr>
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
    <tr key={image.field}>
      <td>
        <Code>{image.field}</Code>
      </td>
      <td>{image.type}</td>
      <td>{image.description}</td>
    </tr>
  ));

  // post resource table
  const posts = [
    { field: "id", type: "integer", description: "unique identifier" },
    {
      field: "title",
      type: "string",
      description: "title of post",
    },
    {
      field: "author",
      type: "string",
      description: "author of post",
    },
    {
      field: "permalink",
      type: "string",
      description: "url of post source",
    },
    {
      field: "score",
      type: "integer",
      description: "total votes of post",
    },
    {
      field: "timestamp",
      type: "integer",
      description: "time of post creation (unix time)",
    },
    {
      field: "nsfw",
      type: "bool",
      description: "image is NSFW (not safe for work, 18+)",
    },
    {
      field: "grayscale",
      type: "bool",
      description: "image is graysacle (black & white)",
    },
    {
      field: "sprocket",
      type: "bool",
      description: "image is a sprocket shot (exposed film sockets)",
    },
    {
      field: "images",
      type: "array[image]",
      description: "list of image",
    },
  ];

  const postRows = posts.map((post) => (
    <tr key={post.field}>
      <td>
        <Code>{post.field}</Code>
      </td>
      <td>{post.type}</td>
      <td>{post.description}</td>
    </tr>
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
  ];

  const metaRows = metas.map((meta) => (
    <tr key={meta.field}>
      <td>
        <Code>{meta.field}</Code>
      </td>
      <td>{meta.type}</td>
      <td>{meta.description}</td>
    </tr>
  ));

  // general params table
  const generals = [
    {
      param: "sort",
      description: "how to order the posts",
      default: "latest",
      options: "latest, top, random",
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
  ];

  const generalRows = generals.map((general) => (
    <tr key={general.param}>
      <td>
        <Code>{general.param}</Code>
      </td>
      <td>{general.description}</td>
      <td>{general.default}</td>
      <td>{general.options}</td>
    </tr>
  ));

  // filters params table
  const filters = [
    {
      param: "nsfw",
      description: "include nsfw (18+) images",
    },
    {
      param: "grayscale",
      description: "include grayscale (black & white) images",
    },
    {
      param: "sprocket",
      description: "include sprocket images",
    },
  ];

  const filterRows = filters.map((filter) => (
    <tr key={filter.param}>
      <td>
        <Code>{filter.param}</Code>
      </td>
      <td>{filter.description}</td>
    </tr>
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
            <Code>https://api.analogdb.com</Code>
          </p>
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
          <h1 className={styles.h1}> Rate Limiting </h1>
          <p>
            The Analogdb API currently places a limit of 30 requests/min.
            Current rate limit status is returned in response headers after each
            request.
          </p>
          <Code block>
            X-Ratelimit-Limit: 30
            <br></br>X-Ratelimit-Remaining: 29
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
            <thead>
              <tr>
                <th>param</th>
                <th>description</th>
              </tr>
            </thead>
            <tbody>{paginationRows}</tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <Prism
                copyLabel="copy example"
                copiedLabel="copied"
                language="yaml"
                styles={() => ({
                  code: {
                    fontSize: "0.75rem",
                  },
                })}
              >
                curl https://api.analogdb.com/posts?page_size=10&page_id=774
              </Prism>
            </div>
          )}
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
          <h1 className={styles.h1}> Resources </h1>
          <h2 className={styles.h2}> Image </h2>
          <p>
            The <Code>image</Code> resource contains the URL of the image as
            well as information about the resolution.
          </p>
          <Table highlightOnHover withColumnBorders>
            <thead>
              <tr>
                <th>field name</th>
                <th>type</th>
                <th>description</th>
              </tr>
            </thead>
            <tbody>{imageRows}</tbody>
          </Table>
          <h2 className={styles.h2}> Post </h2>
          <p>
            The <Code>post</Code> resource contains a list of <Code>image</Code>
            (same picture, multiple resolutions) as well as metadata about the
            post (title, author, etc).
          </p>
          <Table highlightOnHover withColumnBorders>
            <thead>
              <tr>
                <th>field name</th>
                <th>type</th>
                <th>description</th>
              </tr>
            </thead>
            <tbody>{postRows}</tbody>
          </Table>
          <h2 className={styles.h2}> Meta </h2>
          <p>
            The <Code>meta</Code> resource contains supplementary information
            about a collection of <Code>post</Code> resources.
          </p>
          <Table highlightOnHover withColumnBorders>
            <thead>
              <tr>
                <th>field name</th>
                <th>type</th>
                <th>description</th>
              </tr>
            </thead>
            <tbody>{metaRows}</tbody>
          </Table>
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
          <h1 className={styles.h1}> Endpoints </h1>
          <h2 className={styles.h2}> /posts </h2>
          <p>
            Returns a collection of <Code>post</Code> resources with
            accompanying <Code>meta</Code> resource.
          </p>
          <h3 className={styles.h3}>Query Parameters</h3>
          <p>
            Posts can be sorted by time, score, or pseudo-randomly. Limits can
            be placed for maximum number of returned posts. If total number of
            posts exceeds the limit, results will be paginated.
          </p>
          <Table highlightOnHover withColumnBorders>
            <thead>
              <tr>
                <th>param</th>
                <th>description</th>
                <th>default</th>
                <th>options</th>
              </tr>
            </thead>
            <tbody>{generalRows}</tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <Prism
                copyLabel="copy example"
                copiedLabel="copied"
                language="yaml"
                styles={() => ({
                  code: {
                    fontSize: "0.75rem",
                  },
                })}
              >
                curl https://api.analogdb.com/posts?sort=top&page_size=50
              </Prism>
            </div>
          )}
          <p>
            Additionally, posts can be filtered to include, exclude or only
            return grayscale, nsfw and sprocket images.
          </p>
          <p>
            If filter query parameters are not provided (null), posts of that
            type are included in response. If filter is set to <Code>true</Code>{" "}
            only photos of that type are returned. If set to <Code>false</Code>{" "}
            , posts of that type are excluded.
          </p>
          <Table highlightOnHover withColumnBorders>
            <thead>
              <tr>
                <th>param</th>
                <th>description</th>
              </tr>
            </thead>
            <tbody>{filterRows}</tbody>
          </Table>
          {!isMobile && (
            <div className={styles.codeblock}>
              <Prism
                copyLabel="copy example"
                copiedLabel="copied"
                language="yaml"
                styles={() => ({
                  code: {
                    fontSize: "0.75rem",
                  },
                })}
              >
                curl
                https://api.analogdb.com/posts?grayscale=true&sprocket=true&nsfw=false
              </Prism>
            </div>
          )}
          <h2 className={styles.h2}> /post/:id </h2>
          <p>
            Returns a single specific <Code>post</Code> resource as identified
            by ID.
          </p>
          {!isMobile && (
            <div className={styles.codeblock}>
              <Prism
                copyLabel="copy example"
                copiedLabel="copied"
                language="yaml"
                styles={() => ({
                  code: {
                    fontSize: "0.75rem",
                  },
                })}
              >
                curl https://api.analogdb.com/post/1924
              </Prism>
            </div>
          )}
          <div className={styles.divider}>
            <Divider my="sm" />
          </div>
        </div>
      </div>
      <Footer />
    </main>
  );
}
