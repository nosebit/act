const config = {
  branches: ["main"],
  plugins: [
    ["@semantic-release/commit-analyzer", {
      "releaseRules": [
        {"type": "docs", "release": "patch"},
        {"type": "refactor", "release": "patch"},
        {"type": "style", "release": "patch"}
      ]
    }]
  ]
};

if (process.env.VERSION_ONLY) {
  config.plugins = [
    ...config.plugins,
    ["@semantic-release/exec", {
      "verifyReleaseCmd": "echo ${nextRelease.version} > version"
    }]
  ];
} else {
  config.plugins = [
    ...config.plugins,
    "@semantic-release/release-notes-generator",
    "@semantic-release/changelog",
    ["@google/semantic-release-replace-plugin", {
      "replacements": [
        {
          "files": ["README.md"],
          "from": "https://github.com/nosebit/act/releases/download/[^/]+/act-[^-]+",
          "to": "https://github.com/nosebit/act/releases/download/v${nextRelease.version}/act-${nextRelease.version}",
          "countMatches": true
        }
      ]
    }],
    ["@semantic-release/git", {
      "assets": ["README.md"],
      "message": "chore(release): ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}"
    }],
    ["@semantic-release/github", {
      "assets": ".releases/**/*.tar.gz"
    }]
  ]
}

module.exports = config;
