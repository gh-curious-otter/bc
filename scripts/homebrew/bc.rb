# Homebrew formula for bc
# To use: brew tap rpuneet/tap && brew install bc
# Or copy this file to your homebrew-tap repository

class Bc < Formula
  desc "CLI-first orchestration system for AI agent teams"
  homepage "https://github.com/rpuneet/bc"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/rpuneet/bc/releases/download/v#{version}/bc-darwin-amd64"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64"
    end

    on_arm do
      url "https://github.com/rpuneet/bc/releases/download/v#{version}/bc-darwin-arm64"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/rpuneet/bc/releases/download/v#{version}/bc-linux-amd64"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end

    on_arm do
      url "https://github.com/rpuneet/bc/releases/download/v#{version}/bc-linux-arm64"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    end
  end

  depends_on "tmux"

  def install
    binary_name = "bc-#{OS.kernel_name.downcase}-#{Hardware::CPU.arch == :arm64 ? "arm64" : "amd64"}"
    bin.install binary_name => "bc"
  end

  test do
    assert_match "bc version", shell_output("#{bin}/bc version")
  end
end
