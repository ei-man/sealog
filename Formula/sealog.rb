class Sealog < Formula
  desc "Local structured log viewer for stdin and Azure Application Insights"
  homepage "https://github.com/ei-man/sealog"
  license "AGPL-3.0-only"
  head "https://github.com/ei-man/sealog.git", branch: "main"

  depends_on "go" => :build
  depends_on "node" => :build

  def install
    system "npm", "--prefix", "frontend", "ci"
    system "npm", "--prefix", "frontend", "run", "build"

    sealog_version = build.head? ? "HEAD" : version.to_s
    system "go", "build",
      "-ldflags", "-s -w -X main.version=#{sealog_version}",
      "-o", bin/"sealog",
      "."
  end

  test do
    assert_match "sealog", shell_output("#{bin}/sealog --version")
  end
end
