import Footer from "@components/footer";
import Header from "@components/header";
import { logoutAction } from "@lib/auth";
import { Button, Container, Group, Text, Title } from "@mantine/core";
import { IconLogout } from "@tabler/icons-react";
import styles from "./adminPanel.module.css";

export default function AdminPanel(props) {
  let isAdmin = props.isAdmin;
  return (
    <div className={styles.main}>
      <Header isAdmin={isAdmin} />
      <Container size="xl" className={styles.container}>
        <div className={styles.wrapper}>
          <Group justify="space-between" align="center" mb="xl">
            <Title order={1} c="gray.9">
              Admin Panel
            </Title>

            <form action={logoutAction}>
              <Button
                type="submit"
                color="red"
                variant="outline"
                leftSection={<IconLogout size={16} />}
              >
                Logout
              </Button>
            </form>
          </Group>

          <div className={styles.contentCard}>
            <div className={styles.cardHeader}>
              <Title order={3} mb="xs">
                Welcome to Admin Dashboard
              </Title>
              <Text size="sm" c="dimmed">
                You are logged in as an administrator.
              </Text>
            </div>
          </div>
        </div>
      </Container>
      <Footer />
    </div>
  );
}
