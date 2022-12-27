import Masonry from "react-responsive-masonry";
import { useBreakpoint } from "../providers/breakpoint.js";
import GridImage from "./gridImage";

export default function Grid(props) {
  const breakpoints = useBreakpoint();

  let numColumn = 4;
  if (breakpoints["xs"]) {
    numColumn = 1;
  } else if (breakpoints["sm"]) {
    numColumn = 2;
  } else if (breakpoints["md"]) {
    numColumn = 3;
  } else if (breakpoints["lg"]) {
    numColumn = 4;
  }

  return (
    <Masonry columnsCount={numColumn} gutter={"15px"}>
      {props.posts.map((post, index) => (
        <div key={index}>
          <GridImage post={post}></GridImage>
        </div>
      ))}
    </Masonry>
  );
}
