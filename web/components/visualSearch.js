"use client";

import {
  imageAccept,
  imageMaxSize,
  rejectMessage,
} from "@lib/imageSearchStore";
import { Dropzone } from "@mantine/dropzone";
import { IconPhotoScan, IconPhotoX, IconUpload } from "@tabler/icons-react";
import Image from "next/image";
import styles from "./visualSearch.module.css";

export default function VisualSearch({
  examples = [],
  loading,
  error,
  onImage,
  onError,
  onExample,
}) {
  return (
    <div className={styles.panel}>
      <h3 className={styles.title}>visual search</h3>
      <Dropzone
        onDrop={(files) => files[0] && onImage(files[0])}
        onReject={(rejections) => onError(rejectMessage(rejections))}
        accept={imageAccept}
        maxSize={imageMaxSize}
        multiple={false}
        loading={loading}
        radius="md"
        className={styles.zone}
      >
        <div className={styles.zoneInner}>
          <Dropzone.Accept>
            <IconUpload size={32} stroke={1.5} />
          </Dropzone.Accept>
          <Dropzone.Reject>
            <IconPhotoX size={32} stroke={1.5} />
          </Dropzone.Reject>
          <Dropzone.Idle>
            <IconPhotoScan size={32} stroke={1.5} />
          </Dropzone.Idle>
          <p className={styles.zoneText}>
            drag and drop an image here or{" "}
            <span className={styles.browse}>browse</span>
          </p>
          <p className={styles.zoneHint}>
            or paste an image from your clipboard
          </p>
        </div>
      </Dropzone>
      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
      {examples.length > 0 && (
        <>
          <p className={styles.examplesTitle}>or try one of these</p>
          <div className={styles.examples}>
            {examples.map((example) => (
              <button
                type="button"
                key={example.id}
                className={styles.example}
                style={{ backgroundColor: example.image.hex }}
                onClick={() => onExample(example.id)}
                disabled={loading}
                aria-label={`find photos similar to ${example.title}`}
              >
                <Image
                  src={example.image.url}
                  alt=""
                  fill
                  sizes="120px"
                  style={{ objectFit: "cover" }}
                />
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
