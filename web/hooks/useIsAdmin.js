import { useEffect, useState } from "react";

const adminHintCookie = "admin-hint";

const useIsAdmin = () => {
  const [isAdmin, setIsAdmin] = useState(false);

  useEffect(() => {
    const hasHint = document.cookie
      .split(";")
      .some((cookie) => cookie.trim() === `${adminHintCookie}=1`);
    setIsAdmin(hasHint);
  }, []);

  return isAdmin;
};

export default useIsAdmin;
