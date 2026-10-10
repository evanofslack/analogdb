import { useEffect, useRef, useState } from "react";

const threshold = 8;

const useScrollUp = (ref, { paused = false } = {}) => {
  const [shown, setShown] = useState(false);
  const pausedRef = useRef(paused);
  pausedRef.current = paused;

  useEffect(() => {
    const max = () =>
      document.documentElement.scrollHeight - window.innerHeight;
    const position = () => Math.min(Math.max(window.scrollY, 0), max());
    let last = position();
    let frame = null;

    const update = () => {
      frame = null;
      const y = position();
      if (y <= 0) {
        setShown(false);
        last = y;
        return;
      }
      if (Math.abs(y - last) < threshold) return;
      if (y < last) {
        const element = ref.current;
        const bottom = element
          ? element.getBoundingClientRect().bottom + window.scrollY
          : 0;
        if (y > bottom) setShown(true);
      } else if (!pausedRef.current) {
        setShown(false);
      }
      last = y;
    };

    const onScroll = () => {
      if (frame === null) frame = requestAnimationFrame(update);
    };

    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      if (frame !== null) cancelAnimationFrame(frame);
    };
  }, [ref]);

  return shown;
};

export default useScrollUp;
