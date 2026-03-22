import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { Dashboard } from "../Dashboard";
import { Agents } from "../Agents";
import { Channels } from "../Channels";
import { Costs } from "../Costs";
import { Roles } from "../Roles";
import { Tools } from "../Tools";
import { MCP } from "../MCP";
import { Logs } from "../Logs";
import { Doctor } from "../Doctor";
import { Cron } from "../Cron";
import { Secrets } from "../Secrets";
import { Workspace } from "../Workspace";

const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;

function wrap(ui: React.ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

function jsonResponse(body: unknown) {
  return Promise.resolve({
    ok: true,
    status: 200,
    statusText: "OK",
    json: () => Promise.resolve(body),
  } as Response);
}

beforeEach(() => {
  fetchMock.mockReset();
});

function expectSkeletonLoading(container: HTMLElement) {
  const pulseElements = container.querySelectorAll(".animate-pulse");
  expect(pulseElements.length).toBeGreaterThan(0);
}

describe("Dashboard", () => {
  it("renders skeleton loading then data", async () => {
    fetchMock.mockImplementation((url: string) => {
      if (url.includes("/agents")) return jsonResponse([]);
      if (url.includes("/channels")) return jsonResponse([]);
      if (url.includes("/costs"))
        return jsonResponse({
          input_tokens: 0,
          output_tokens: 0,
          total_tokens: 100,
          total_cost_usd: 1.5,
          record_count: 2,
        });
      return jsonResponse({});
    });
    const { container } = wrap(<Dashboard />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
    });
  });

  it("renders empty state for no agents", async () => {
    fetchMock.mockImplementation((url: string) => {
      if (url.includes("/agents")) return jsonResponse([]);
      if (url.includes("/channels")) return jsonResponse([]);
      if (url.includes("/costs"))
        return jsonResponse({
          input_tokens: 0,
          output_tokens: 0,
          total_tokens: 0,
          total_cost_usd: 0,
          record_count: 0,
        });
      return jsonResponse({});
    });
    wrap(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("No agents running")).toBeInTheDocument();
    });
  });
});

describe("Agents", () => {
  it("renders skeleton loading then agent list", async () => {
    fetchMock.mockReturnValue(
      jsonResponse([
        {
          name: "bot-1",
          role: "engineer",
          tool: "claude",
          state: "running",
          cost_usd: 0.01,
          started_at: "",
        },
      ]),
    );
    const { container } = wrap(<Agents />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("bot-1")).toBeInTheDocument();
    });
  });
});

describe("Channels", () => {
  it("renders skeleton loading then channel list", async () => {
    fetchMock.mockReturnValue(
      jsonResponse([
        { name: "general", description: "", members: [], member_count: 3 },
      ]),
    );
    const { container } = wrap(<Channels />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("#general")).toBeInTheDocument();
    });
  });
});

describe("Costs", () => {
  it("renders skeleton loading then cost data", async () => {
    fetchMock.mockImplementation((url: string) => {
      if (url.includes("/costs/budgets")) return jsonResponse([]);
      if (url.includes("/costs/models")) return jsonResponse([]);
      if (url.includes("/costs/daily")) return jsonResponse([]);
      if (url.includes("/costs/agents")) return jsonResponse([]);
      if (url.includes("/costs"))
        return jsonResponse({
          input_tokens: 0,
          output_tokens: 0,
          total_tokens: 0,
          total_cost_usd: 0,
          record_count: 0,
        });
      return jsonResponse({});
    });
    const { container } = wrap(<Costs />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("Costs")).toBeInTheDocument();
    });
  });
});

describe("Roles", () => {
  it("renders skeleton loading then role cards", async () => {
    fetchMock.mockReturnValue(
      jsonResponse({
        eng: {
          Name: "engineer",
          Prompt: "",
          MCPServers: [],
          Secrets: [],
          Plugins: [],
          PromptCreate: "",
          PromptStart: "",
          PromptStop: "",
          PromptDelete: "",
          Commands: {},
          Skills: {},
          Agents: {},
          Rules: {},
          Settings: {},
          Review: "",
        },
      }),
    );
    const { container } = wrap(<Roles />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("engineer")).toBeInTheDocument();
    });
  });
});

describe("Tools", () => {
  it("renders skeleton loading then tool table", async () => {
    fetchMock.mockReturnValue(
      jsonResponse([
        {
          name: "my-tool",
          command: "/usr/bin/tool",
          install_cmd: "",
          builtin: true,
          enabled: true,
        },
      ]),
    );
    const { container } = wrap(<Tools />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("my-tool")).toBeInTheDocument();
    });
  });
});

describe("MCP", () => {
  it("renders skeleton loading then server list", async () => {
    fetchMock.mockReturnValue(
      jsonResponse([
        {
          name: "test-server",
          transport: "stdio",
          command: "node",
          url: "",
          enabled: true,
        },
      ]),
    );
    const { container } = wrap(<MCP />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("test-server")).toBeInTheDocument();
    });
  });
});

describe("Logs", () => {
  it("renders skeleton loading then event log", async () => {
    fetchMock.mockReturnValue(
      jsonResponse([
        {
          id: 1,
          type: "agent.start",
          agent: "bot",
          message: "started",
          created_at: "2025-01-01T00:00:00Z",
        },
      ]),
    );
    const { container } = wrap(<Logs />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("Event Log")).toBeInTheDocument();
    });
  });

  it("renders empty state when no logs", async () => {
    fetchMock.mockReturnValue(jsonResponse([]));
    wrap(<Logs />);
    await waitFor(() => {
      expect(screen.getByText("No events recorded yet")).toBeInTheDocument();
    });
  });
});

describe("Doctor", () => {
  it("renders skeleton loading then report", async () => {
    fetchMock.mockReturnValue(
      jsonResponse({
        Categories: [
          {
            Name: "System",
            Items: [{ Name: "go", Message: "installed", Fix: "", Severity: 0 }],
          },
        ],
      }),
    );
    const { container } = wrap(<Doctor />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("Doctor")).toBeInTheDocument();
    });
  });
});

describe("Cron", () => {
  it("renders skeleton loading then cron table", async () => {
    fetchMock.mockReturnValue(
      jsonResponse([
        {
          name: "nightly",
          schedule: "0 0 * * *",
          agent_name: "bot",
          prompt: "",
          command: "",
          enabled: true,
          run_count: 5,
          last_run: null,
          next_run: null,
          created_at: "",
        },
      ]),
    );
    const { container } = wrap(<Cron />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("Cron Jobs")).toBeInTheDocument();
      expect(screen.getByText(/nightly/)).toBeInTheDocument();
    });
  });
});

describe("Secrets", () => {
  it("renders skeleton loading then secrets table", async () => {
    fetchMock.mockReturnValue(
      jsonResponse([
        {
          name: "API_KEY",
          description: "key",
          backend: "env",
          created_at: "2025-01-01",
        },
      ]),
    );
    const { container } = wrap(<Secrets />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("API_KEY")).toBeInTheDocument();
    });
  });
});

describe("Workspace", () => {
  it("renders skeleton loading then workspace status", async () => {
    fetchMock.mockReturnValue(
      jsonResponse({ root_dir: "/home/project", version: "2" }),
    );
    const { container } = wrap(<Workspace />);
    expectSkeletonLoading(container);
    await waitFor(() => {
      expect(screen.getByText("Workspace")).toBeInTheDocument();
    });
  });
});
