"use client";

import { Button, Grid, Paper, Text, TextInput } from "@mantine/core";
import { IconSearch, IconTrendingUp } from "@tabler/icons-react";
import { useState } from "react";
import styles from "./search.module.css";

export default function Search({
  textTemp,
  setTextTemp,
  textPlaceholder,
  onSearch,
  onClose,
}) {
  const [typedSearch, setTypedSearch] = useState(textTemp || "");

  const trendingSearches = [
    "beach",
    "baseball",
    "moonrise",
    "atlantic",
    "cancun",
    "neon",
    "new york",
    "sunset",
  ];

  const handleKeyDown = (event) => {
    if (event.key === "Enter" && typedSearch.trim()) {
      setTextTemp(typedSearch.trim());
      onSearch(typedSearch.trim());
      onClose();
    }
    if (event.key === "Escape") {
      onClose();
    }
  };

  const handleTrendingClick = (searchTerm) => {
    setTextTemp(searchTerm);
    onSearch(searchTerm);
    onClose();
  };

  const handleSearchClick = () => {
    if (typedSearch.trim()) {
      setTextTemp(typedSearch.trim());
      onSearch(typedSearch.trim());
      onClose();
    }
  };

  return (
    <div className={styles.searchContainer}>
      <div className={styles.searchInput}>
        <TextInput
          size="md"
          placeholder={textPlaceholder}
          leftSection={<IconSearch size={20} stroke={1.5} />}
          value={typedSearch}
          onChange={(event) => setTypedSearch(event.currentTarget.value)}
          onKeyDown={handleKeyDown}
          style={{ flex: 1 }}
          autoFocus
        />
        <Button
          variant="filled"
          size="md"
          onClick={handleSearchClick}
          disabled={!typedSearch.trim()}
        >
          search
        </Button>
      </div>

      <div className={styles.trendingSection}>
        <div className={styles.trendingHeader}>
          <IconTrendingUp size={18} stroke={1.5} />
          <Text size="sm" fw={500} c="dimmed">
            trending searches
          </Text>
        </div>

        <Grid className={styles.trendingGrid}>
          {trendingSearches.map((search, index) => (
            <Grid.Col span={6} key={index}>
              <Paper
                className={styles.trendingItem}
                onClick={() => handleTrendingClick(search)}
              >
                <Text size="sm">{search}</Text>
              </Paper>
            </Grid.Col>
          ))}
        </Grid>
      </div>
    </div>
  );
}
