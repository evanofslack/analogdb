'use client';

import styles from './deletePostButton.module.css';

export default function DeletePostButton({ postId }) {
  return (
    <button onClick={handleDelete} className={styles.deleteButton}>
      Delete Post
    </button>
  );
}
