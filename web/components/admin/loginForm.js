import Footer from "@components/footer";
import Header from "@components/header";
import { loginAction } from "@lib/auth";
import {
  Alert,
  Button,
  Paper,
  PasswordInput,
  Stack,
  TextInput,
  Title,
} from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import styles from "./loginForm.module.css";

export default function LoginForm(props) {
  let error = props.error;
  let isAdmin = props.isAdmin;
  return (
    <div className={styles.main}>
      <Header isAdmin={isAdmin} />
      <div className={styles.container}>
        <Paper
          className={styles.formWrapper}
          shadow="sm"
          radius="md"
          withBorder
        >
          <div className={styles.header}>
            <Title order={2} className={styles.title}>
              Admin Login
            </Title>
          </div>

          <form action={loginAction}>
            <Stack gap="lg">
              <TextInput
                id="username"
                name="username"
                label="Username"
                placeholder="admin username"
                required
                size="sm"
              />

              <PasswordInput
                id="password"
                name="password"
                label="Password"
                placeholder="admin password"
                required
                size="sm"
              />

              {error === "invalid" && (
                <Alert
                  variant="outline"
                  color="red"
                  title="Error"
                  icon={<IconAlertCircle size={16} />}
                >
                  Invalid username or password
                </Alert>
              )}

              <Button type="submit" fullWidth variant="filled" size="sm">
                Sign In
              </Button>
            </Stack>
          </form>
        </Paper>
      </div>
      <Footer />
    </div>
  );
}
