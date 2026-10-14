"use client";

import { ApiDemo, ApiDemoGroup, ApiDemoLayout, shuffle } from "@lib/about";
import { publicURL } from "@lib/constants";
import { CodeHighlight } from "@mantine/code-highlight";
import { IconCheck, IconCopy } from "@tabler/icons-react";
import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import styles from "./aboutApi.module.css";
import AboutPhoto from "./aboutPhoto";

const TYPE_MS = 30;
const DELETE_MS = 18;
const WAIT_MS = 400;
const LINE_MS = 35;
const HOLD_MS = 5000;
const COPIED_MS = 1500;

// how many photos each layout shows
const SLOTS = { post: 1, grid: 3, similar: 5 };

type Phase = "typing" | "printing" | "holding";

interface Shown {
  layout: ApiDemoLayout;
  demo: ApiDemo;
}

const command = (demo: ApiDemo) => `curl ${publicURL}${demo.path}`;

// long urls wrap after a slash, question mark or ampersand, not mid word
function breakable(text: string) {
  return text.split(/(?<=[/?&])/).map((part, i) => (
    <Fragment key={i}>
      {i > 0 && <wbr />}
      {part}
    </Fragment>
  ));
}

export default function AboutApi({ groups }: { groups: ApiDemoGroup[] }) {
  const [variants, setVariants] = useState(() => groups.map((g) => g.demos));
  const [index, setIndex] = useState(0);
  const [picks, setPicks] = useState(() => groups.map(() => 0));
  const [shown, setShown] = useState<Shown | null>(null);
  const [phase, setPhase] = useState<Phase>("typing");
  const [text, setText] = useState("");
  const [lines, setLines] = useState(0);
  const [reduced, setReduced] = useState(false);
  const [visible, setVisible] = useState(false);
  const [hovered, setHovered] = useState(false);
  const [full, setFull] = useState(false);
  const [copied, setCopied] = useState(false);
  const ref = useRef<HTMLDivElement | null>(null);

  const current = variants[index]?.[picks[index]];

  const shortLines = useMemo(
    () => (shown ? shown.demo.short.split("\n") : []),
    [shown]
  );

  // show a request and its whole response at once
  const show = (next: number, nextPicks: number[], list = variants) => {
    const demo = list[next][nextPicks[next]];
    setPicks(nextPicks);
    setIndex(next);
    setShown({ layout: groups[next].layout, demo });
    setText(command(demo));
    setLines(demo.short.split("\n").length);
    setPhase("holding");
  };

  // leaving an endpoint moves it on, so the next visit shows another request
  const advance = (list: number[]) =>
    list.map((pick, g) =>
      g === index ? (pick + 1) % variants[g].length : pick
    );

  useEffect(() => {
    const shuffled = groups.map((g) => shuffle(g.demos));
    setVariants(shuffled);
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      setReduced(true);
      show(
        0,
        groups.map(() => 0),
        shuffled
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groups]);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    const observer = new IntersectionObserver(([entry]) =>
      setVisible(entry.isIntersecting)
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  // delete back to what the next request shares, type the rest, wait on the network,
  // print the response, hold, then move to the next endpoint
  useEffect(() => {
    if (reduced || !visible || hovered || full || !current) return;
    let timer: ReturnType<typeof setTimeout>;
    if (phase === "typing") {
      const target = command(current);
      if (!target.startsWith(text)) {
        timer = setTimeout(() => setText((t) => t.slice(0, -1)), DELETE_MS);
      } else if (text.length < target.length) {
        timer = setTimeout(
          () => setText(target.slice(0, text.length + 1)),
          TYPE_MS
        );
      } else {
        timer = setTimeout(() => {
          setShown({ layout: groups[index].layout, demo: current });
          setLines(0);
          setPhase("printing");
        }, WAIT_MS);
      }
    } else if (phase === "printing") {
      timer =
        lines < shortLines.length
          ? setTimeout(() => setLines((n) => n + 1), LINE_MS)
          : setTimeout(() => setPhase("holding"), 0);
    } else {
      timer = setTimeout(() => {
        setPicks(advance);
        setIndex((index + 1) % groups.length);
        setPhase("typing");
      }, HOLD_MS);
    }
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    groups,
    current,
    shortLines,
    reduced,
    visible,
    hovered,
    full,
    phase,
    index,
    text,
    lines,
  ]);

  const fullJSON = useMemo(
    () => (shown ? JSON.stringify(shown.demo.response, null, 2) : ""),
    [shown]
  );

  if (!current) return null;

  const toggleFull = () => {
    if (!full) show(index, picks);
    setFull(!full);
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(command(current));
      setCopied(true);
      setTimeout(() => setCopied(false), COPIED_MS);
    } catch {
      // clipboard can be blocked, the command is still on screen to select
    }
  };

  // the last response stays dimmed while the next request types
  const stale = phase === "typing" && shown !== null;
  const layout = shown?.layout ?? groups[index].layout;
  const code = !shown
    ? ""
    : full
    ? fullJSON
    : shortLines.slice(0, lines).join("\n");

  return (
    <div
      ref={ref}
      className={styles.card}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <div className={styles.request}>
        <span className={styles.prompt} aria-hidden>
          $
        </span>
        {/* every command sits in one cell, so the line keeps the height of the longest */}
        <div className={styles.command}>
          {groups.flatMap((group) =>
            group.demos.map((demo) => (
              <span key={demo.path} className={styles.ghost} aria-hidden>
                {breakable(command(demo))}
              </span>
            ))
          )}
          <span>
            {breakable(text)}
            {!reduced && <span className={styles.caret} aria-hidden />}
          </span>
        </div>
        <button
          type="button"
          className={styles.copy}
          onClick={copy}
          aria-label="copy request"
        >
          {copied ? (
            <IconCheck size={16} stroke={1.5} />
          ) : (
            <IconCopy size={16} stroke={1.5} />
          )}
        </button>
      </div>

      <div className={`${styles.body} ${styles[layout]}`}>
        <div
          className={`${styles.response} ${full ? styles.full : ""} ${
            stale ? styles.stale : ""
          }`}
        >
          {code && (
            <CodeHighlight
              code={code}
              language="json"
              withCopyButton={full}
              classNames={{
                codeHighlight: styles.highlight,
                code: styles.code,
              }}
            />
          )}
        </div>
        <div className={`${styles.photos} ${stale ? styles.stale : ""}`}>
          {shown
            ? shown.demo.photos
                .slice(0, SLOTS[layout])
                .map((photo) => (
                  <AboutPhoto
                    key={`${shown.demo.path}-${photo.id}`}
                    photo={photo}
                    small
                    fill
                    sizes="(max-width: 720px) 33vw, 240px"
                    className={`${styles.tile} ${styles.fadeIn}`}
                  />
                ))
            : Array.from({ length: SLOTS[layout] }, (_, i) => (
                <div key={i} className={`${styles.tile} ${styles.empty}`} />
              ))}
        </div>
      </div>

      <div className={styles.footer}>
        <div className={styles.steps}>
          {groups.map((group, i) => (
            <button
              key={group.name}
              type="button"
              className={`${styles.step} ${i === index ? styles.active : ""}`}
              onClick={() => show(i, advance(picks))}
              aria-current={i === index}
            >
              {group.name}
            </button>
          ))}
        </div>
        <button
          type="button"
          className={styles.step}
          onClick={toggleFull}
          aria-pressed={full}
        >
          {full ? "short response" : "full response"}
        </button>
      </div>
    </div>
  );
}
