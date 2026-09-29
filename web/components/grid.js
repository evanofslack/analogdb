"use client";

import { useBreakpoint } from "@providers/breakpoint.js";
import Masonry from "react-responsive-masonry";
import GridImage from "./gridImage";

export default function Grid(props) {
  const breakpoints = useBreakpoint();

  let numColumn = 4;
  if (breakpoints["xs"]) {
    numColumn = 2;
  } else if (breakpoints["sm"]) {
    numColumn = 2;
  } else if (breakpoints["md"]) {
    numColumn = 3;
  } else if (breakpoints["lg"]) {
    numColumn = 3;
  } else if (breakpoints["xl"]) {
    numColumn = 4;
  }

  return (
    <Masonry columnsCount={numColumn} gutter={"15px"}>
      {props.posts.map((post) => (
        // Masonry wraps items in a flex row, so without a width they shrink to
        // fit and unloaded images collapse to 0x0
        <div key={post.id} style={{ width: "100%" }}>
          <GridImage post={post}></GridImage>
        </div>
      ))}
    </Masonry>
  );
}
