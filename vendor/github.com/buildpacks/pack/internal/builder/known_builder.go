package builder

type KnownBuilder struct {
	Vendor             string
	Image              string
	DefaultDescription string
	Suggested          bool
	Trusted            bool
}

var KnownBuilders = []KnownBuilder{
	{
		Vendor:             "Google",
		Image:              "gcr.io/buildpacks/builder:google-22",
		DefaultDescription: "Ubuntu 22.04 base image with buildpacks for .NET, Dart, Go, Java, Node.js, PHP, Python, and Ruby",
		Suggested:          true,
		Trusted:            true,
	},
	{
		Vendor:             "Heroku",
		Image:              "heroku/builder:24",
		DefaultDescription: "Ubuntu 24.04 AMD64+ARM64 base image with buildpacks for Go, Java, Node.js, PHP, Python, Ruby & Scala.",
		Suggested:          true,
		Trusted:            true,
	},
	{
		Vendor:             "Heroku",
		Image:              "heroku/builder:22",
		DefaultDescription: "Ubuntu 22.04 AMD64 base image with buildpacks for Go, Java, Node.js, PHP, Python, Ruby & Scala.",
		Suggested:          false,
		Trusted:            true,
	},
	{
		Vendor:             "Heroku",
		Image:              "heroku/builder:20",
		DefaultDescription: "Ubuntu 20.04 AMD64 base image with buildpacks for Go, Java, Node.js, PHP, Python, Ruby & Scala.",
		Suggested:          false,
		Trusted:            true,
	},
	{
		Vendor:             "Paketo Buildpacks",
		Image:              "paketobuildpacks/builder-jammy-base",
		DefaultDescription: "Small base image with buildpacks for Java, Node.js, Golang, .NET Core, Python & Ruby",
		Suggested:          true,
		Trusted:            true,
	},
	{
		Vendor:             "Paketo Buildpacks",
		Image:              "paketobuildpacks/builder-jammy-full",
		DefaultDescription: "Larger base image with buildpacks for Java, Node.js, Golang, .NET Core, Python, Ruby, & PHP",
		Suggested:          true,
		Trusted:            true,
	},
	{
		Vendor:             "Paketo Buildpacks",
		Image:              "paketobuildpacks/builder-jammy-tiny",
		DefaultDescription: "Tiny base image (jammy build image, distroless run image) with buildpacks for Golang & Java",
		Suggested:          true,
		Trusted:            true,
	},
	{
		Vendor:             "Paketo Buildpacks",
		Image:              "paketobuildpacks/builder-jammy-buildpackless-static",
		DefaultDescription: "Static base image (jammy build image, distroless run image) suitable for static binaries like Go or Rust",
		Suggested:          true,
		Trusted:            true,
	},
}

var IsKnownTrustedBuilder = func(b string) bool {
	for _, knownBuilder := range KnownBuilders {
		if b == knownBuilder.Image && knownBuilder.Trusted {
			return true
		}
	}
	return false
}
