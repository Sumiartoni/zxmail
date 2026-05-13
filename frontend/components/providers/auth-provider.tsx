"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { ZxMailApiClient, previewMode } from "@/lib/api";
import type { AuthSession } from "@/types/zxmail";

type AuthContextValue = {
  session: AuthSession | null;
  hydrated: boolean;
  previewMode: boolean;
  login: (email: string, password: string) => Promise<AuthSession>;
  logout: () => Promise<void>;
  api: ZxMailApiClient;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const api = useMemo(() => new ZxMailApiClient(), []);
  const [session, setSession] = useState<AuthSession | null>(null);
  const [hydrated, setHydrated] = useState(previewMode);

  useEffect(() => {
    if (previewMode) {
      return;
    }

    let active = true;

    async function bootstrap() {
      try {
        const user = await api.me();
        if (active) {
          setSession({ user });
        }
      } catch {
        if (active) {
          setSession(null);
        }
      } finally {
        if (active) {
          setHydrated(true);
        }
      }
    }

    void bootstrap()
    return () => {
      active = false;
    };
  }, [api]);

  async function login(email: string, password: string) {
    const nextSession = await api.login(email, password);
    setSession(nextSession);
    return nextSession;
  }

  async function logout() {
    if (!previewMode) {
      await api.logout();
    }
    setSession(null);
  }

  return (
    <AuthContext.Provider
      value={{
        session,
        hydrated,
        previewMode,
        login,
        logout,
        api,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }

  return context;
}
