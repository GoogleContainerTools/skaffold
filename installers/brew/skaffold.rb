class Skaffold < Formula
  desc "A tool that facilitates continuous development for Kubernetes applications."
  url "https://github.com/GoogleCloudPlatform/skaffold.git"
  version "v0.1.0"

  depends_on "go" => :build

  def install
    ENV["GOPATH"] = buildpath
    (buildpath/"src/github.com/GoogleCloudPlatform").mkpath

    ln_s buildpath, buildpath/"src/github.com/GoogleCloudPlatform/skaffold"
    system "make"
    bin.install "out/skaffold"
  end

  test do
    system "#{bin}skaffold", "--version"
  end
end