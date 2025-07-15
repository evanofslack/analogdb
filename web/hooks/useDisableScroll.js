import { useEffect } from "react";

const useDisableScroll = (open) => {
  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "unset";
    }
  }, [open]);
};
export default useDisableScroll;
