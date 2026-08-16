class Anytty < Formula
  desc "Remote terminal, multiplexer, and file workspace for your own machines"
  homepage "https://github.com/anytty/anytty"
  version "0.0.1-beta.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/anytty/anytty/releases/download/v0.0.1-beta.0/anytty-v0.0.1-beta.0-darwin-arm64.tar.gz"
      sha256 "d06c50655efb468181b065566fba985a4e7909c66519d37e9bbcda3654595dbc"
    end
    on_intel do
      url "https://github.com/anytty/anytty/releases/download/v0.0.1-beta.0/anytty-v0.0.1-beta.0-darwin-amd64.tar.gz"
      sha256 "ed16f68262167fadfc55f8576dc0267aca253ffd1a609892aac8f6cf3fc7820b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/anytty/anytty/releases/download/v0.0.1-beta.0/anytty-v0.0.1-beta.0-linux-arm64.tar.gz"
      sha256 "ffc85c79a836e0268972a67924b55085d85b140f301e8afe4df39a0740e1227f"
    end
    on_intel do
      url "https://github.com/anytty/anytty/releases/download/v0.0.1-beta.0/anytty-v0.0.1-beta.0-linux-amd64.tar.gz"
      sha256 "6005dbc56db43f6488c2f434abfb0b9a3ae4bc311e53714c0b96797cb3fb9263"
    end
  end

  def install
    bin.install "anytty"
    prefix.install "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.txt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/anytty --version")
  end
end
