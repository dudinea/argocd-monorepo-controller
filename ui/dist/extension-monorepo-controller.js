(() => {
  var __create = Object.create;
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __getOwnPropSymbols = Object.getOwnPropertySymbols;
  var __getProtoOf = Object.getPrototypeOf;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __propIsEnum = Object.prototype.propertyIsEnumerable;
  var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
  var __spreadValues = (a, b) => {
    for (var prop in b || (b = {}))
      if (__hasOwnProp.call(b, prop))
        __defNormalProp(a, prop, b[prop]);
    if (__getOwnPropSymbols)
      for (var prop of __getOwnPropSymbols(b)) {
        if (__propIsEnum.call(b, prop))
          __defNormalProp(a, prop, b[prop]);
      }
    return a;
  };
  var __commonJS = (cb, mod) => function __require() {
    return mod || (0, cb[__getOwnPropNames(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports;
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
    // If the importer is in node compatibility mode or this is not an ESM
    // file that has been converted to a CommonJS file using a Babel-
    // compatible transform (i.e. "__esModule" has not been set), then set
    // "default" to the CommonJS "module.exports" for node compatibility.
    isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
    mod
  ));

  // node_modules/protocols/lib/index.js
  var require_lib = __commonJS({
    "node_modules/protocols/lib/index.js"(exports, module) {
      "use strict";
      module.exports = function protocols(input, first) {
        if (first === true) {
          first = 0;
        }
        var prots = "";
        if (typeof input === "string") {
          try {
            prots = new URL(input).protocol;
          } catch (e) {
          }
        } else if (input && input.constructor === URL) {
          prots = input.protocol;
        }
        var splits = prots.split(/\:|\+/).filter(Boolean);
        if (typeof first === "number") {
          return splits[first];
        }
        return splits;
      };
    }
  });

  // node_modules/parse-path/lib/index.js
  var require_lib2 = __commonJS({
    "node_modules/parse-path/lib/index.js"(exports, module) {
      "use strict";
      var protocols = require_lib();
      function parsePath(url) {
        var output = {
          protocols: [],
          protocol: null,
          port: null,
          resource: "",
          host: "",
          user: "",
          password: "",
          pathname: "",
          hash: "",
          search: "",
          href: url,
          query: {},
          parse_failed: false
        };
        try {
          var parsed = new URL(url);
          output.protocols = protocols(parsed);
          output.protocol = output.protocols[0];
          output.port = parsed.port;
          output.resource = parsed.hostname;
          output.host = parsed.host;
          output.user = parsed.username || "";
          output.password = parsed.password || "";
          output.pathname = parsed.pathname;
          output.hash = parsed.hash.slice(1);
          output.search = parsed.search.slice(1);
          output.href = parsed.href;
          output.query = Object.fromEntries(parsed.searchParams);
        } catch (e) {
          output.protocols = ["file"];
          output.protocol = output.protocols[0];
          output.port = "";
          output.resource = "";
          output.user = "";
          output.pathname = "";
          output.hash = "";
          output.search = "";
          output.href = url;
          output.query = {};
          output.parse_failed = true;
        }
        return output;
      }
      module.exports = parsePath;
    }
  });

  // node_modules/parse-url/dist/index.js
  var require_dist = __commonJS({
    "node_modules/parse-url/dist/index.js"(exports, module) {
      "use strict";
      var parsePath = require_lib2();
      function _interopDefaultLegacy(e) {
        return e && typeof e === "object" && "default" in e ? e : { "default": e };
      }
      var parsePath__default = /* @__PURE__ */ _interopDefaultLegacy(parsePath);
      var DATA_URL_DEFAULT_MIME_TYPE = "text/plain";
      var DATA_URL_DEFAULT_CHARSET = "us-ascii";
      var testParameter = (name, filters) => filters.some((filter) => filter instanceof RegExp ? filter.test(name) : filter === name);
      var normalizeDataURL = (urlString, { stripHash }) => {
        const match = new RegExp("^data:(?<type>[^,]*?),(?<data>[^#]*?)(?:#(?<hash>.*))?$").exec(urlString);
        if (!match) {
          throw new Error(`Invalid URL: ${urlString}`);
        }
        let { type, data, hash } = match.groups;
        const mediaType = type.split(";");
        hash = stripHash ? "" : hash;
        let isBase64 = false;
        if (mediaType[mediaType.length - 1] === "base64") {
          mediaType.pop();
          isBase64 = true;
        }
        const mimeType = (mediaType.shift() || "").toLowerCase();
        const attributes = mediaType.map((attribute) => {
          let [key, value = ""] = attribute.split("=").map((string) => string.trim());
          if (key === "charset") {
            value = value.toLowerCase();
            if (value === DATA_URL_DEFAULT_CHARSET) {
              return "";
            }
          }
          return `${key}${value ? `=${value}` : ""}`;
        }).filter(Boolean);
        const normalizedMediaType = [
          ...attributes
        ];
        if (isBase64) {
          normalizedMediaType.push("base64");
        }
        if (normalizedMediaType.length > 0 || mimeType && mimeType !== DATA_URL_DEFAULT_MIME_TYPE) {
          normalizedMediaType.unshift(mimeType);
        }
        return `data:${normalizedMediaType.join(";")},${isBase64 ? data.trim() : data}${hash ? `#${hash}` : ""}`;
      };
      function normalizeUrl(urlString, options) {
        options = __spreadValues({
          defaultProtocol: "http:",
          normalizeProtocol: true,
          forceHttp: false,
          forceHttps: false,
          stripAuthentication: true,
          stripHash: false,
          stripTextFragment: true,
          stripWWW: true,
          removeQueryParameters: [/^utm_\w+/i],
          removeTrailingSlash: true,
          removeSingleSlash: true,
          removeDirectoryIndex: false,
          sortQueryParameters: true
        }, options);
        urlString = urlString.trim();
        if (/^data:/i.test(urlString)) {
          return normalizeDataURL(urlString, options);
        }
        if (/^view-source:/i.test(urlString)) {
          throw new Error("`view-source:` is not supported as it is a non-standard protocol");
        }
        const hasRelativeProtocol = urlString.startsWith("//");
        const isRelativeUrl = !hasRelativeProtocol && /^\.*\//.test(urlString);
        if (!isRelativeUrl) {
          urlString = urlString.replace(/^(?!(?:\w+:)?\/\/)|^\/\//, options.defaultProtocol);
        }
        const urlObject = new URL(urlString);
        if (options.forceHttp && options.forceHttps) {
          throw new Error("The `forceHttp` and `forceHttps` options cannot be used together");
        }
        if (options.forceHttp && urlObject.protocol === "https:") {
          urlObject.protocol = "http:";
        }
        if (options.forceHttps && urlObject.protocol === "http:") {
          urlObject.protocol = "https:";
        }
        if (options.stripAuthentication) {
          urlObject.username = "";
          urlObject.password = "";
        }
        if (options.stripHash) {
          urlObject.hash = "";
        } else if (options.stripTextFragment) {
          urlObject.hash = urlObject.hash.replace(/#?:~:text.*?$/i, "");
        }
        if (urlObject.pathname) {
          const protocolRegex = /\b[a-z][a-z\d+\-.]{1,50}:\/\//g;
          let lastIndex = 0;
          let result = "";
          for (; ; ) {
            const match = protocolRegex.exec(urlObject.pathname);
            if (!match) {
              break;
            }
            const protocol2 = match[0];
            const protocolAtIndex = match.index;
            const intermediate = urlObject.pathname.slice(lastIndex, protocolAtIndex);
            result += intermediate.replace(/\/{2,}/g, "/");
            result += protocol2;
            lastIndex = protocolAtIndex + protocol2.length;
          }
          const remnant = urlObject.pathname.slice(lastIndex, urlObject.pathname.length);
          result += remnant.replace(/\/{2,}/g, "/");
          urlObject.pathname = result;
        }
        if (urlObject.pathname) {
          try {
            urlObject.pathname = decodeURI(urlObject.pathname);
          } catch (e) {
          }
        }
        if (options.removeDirectoryIndex === true) {
          options.removeDirectoryIndex = [/^index\.[a-z]+$/];
        }
        if (Array.isArray(options.removeDirectoryIndex) && options.removeDirectoryIndex.length > 0) {
          let pathComponents = urlObject.pathname.split("/");
          const lastComponent = pathComponents[pathComponents.length - 1];
          if (testParameter(lastComponent, options.removeDirectoryIndex)) {
            pathComponents = pathComponents.slice(0, -1);
            urlObject.pathname = pathComponents.slice(1).join("/") + "/";
          }
        }
        if (urlObject.hostname) {
          urlObject.hostname = urlObject.hostname.replace(/\.$/, "");
          if (options.stripWWW && /^www\.(?!www\.)[a-z\-\d]{1,63}\.[a-z.\-\d]{2,63}$/.test(urlObject.hostname)) {
            urlObject.hostname = urlObject.hostname.replace(/^www\./, "");
          }
        }
        if (Array.isArray(options.removeQueryParameters)) {
          for (const key of [...urlObject.searchParams.keys()]) {
            if (testParameter(key, options.removeQueryParameters)) {
              urlObject.searchParams.delete(key);
            }
          }
        }
        if (options.removeQueryParameters === true) {
          urlObject.search = "";
        }
        if (options.sortQueryParameters) {
          urlObject.searchParams.sort();
          try {
            urlObject.search = decodeURIComponent(urlObject.search);
          } catch (e) {
          }
        }
        if (options.removeTrailingSlash) {
          urlObject.pathname = urlObject.pathname.replace(/\/$/, "");
        }
        const oldUrlString = urlString;
        urlString = urlObject.toString();
        if (!options.removeSingleSlash && urlObject.pathname === "/" && !oldUrlString.endsWith("/") && urlObject.hash === "") {
          urlString = urlString.replace(/\/$/, "");
        }
        if ((options.removeTrailingSlash || urlObject.pathname === "/") && urlObject.hash === "" && options.removeSingleSlash) {
          urlString = urlString.replace(/\/$/, "");
        }
        if (hasRelativeProtocol && !options.normalizeProtocol) {
          urlString = urlString.replace(/^http:\/\//, "//");
        }
        if (options.stripProtocol) {
          urlString = urlString.replace(/^(?:https?:)?\/\//, "");
        }
        return urlString;
      }
      var parseUrl = (url, normalize = false) => {
        const GIT_RE = /^(?:([a-z_][a-z0-9_-]{0,31})@|https?:\/\/)([\w\.\-@]+)[\/:]([\~,\.\w,\-,\_,\/]+?(?:\.git|\/)?)$/;
        const throwErr = (msg) => {
          const err = new Error(msg);
          err.subject_url = url;
          throw err;
        };
        if (typeof url !== "string" || !url.trim()) {
          throwErr("Invalid url.");
        }
        if (url.length > parseUrl.MAX_INPUT_LENGTH) {
          throwErr("Input exceeds maximum length. If needed, change the value of parseUrl.MAX_INPUT_LENGTH.");
        }
        if (normalize) {
          if (typeof normalize !== "object") {
            normalize = {
              stripHash: false
            };
          }
          url = normalizeUrl(url, normalize);
        }
        const parsed = parsePath__default["default"](url);
        if (parsed.parse_failed) {
          const matched = parsed.href.match(GIT_RE);
          if (matched) {
            parsed.protocols = ["ssh"];
            parsed.protocol = "ssh";
            parsed.resource = matched[2];
            parsed.host = matched[2];
            parsed.user = matched[1];
            parsed.pathname = `/${matched[3]}`;
            parsed.parse_failed = false;
          } else {
            throwErr("URL parsing failed.");
          }
        }
        return parsed;
      };
      parseUrl.MAX_INPUT_LENGTH = 2048;
      module.exports = parseUrl;
    }
  });

  // node_modules/is-ssh/lib/index.js
  var require_lib3 = __commonJS({
    "node_modules/is-ssh/lib/index.js"(exports, module) {
      "use strict";
      var protocols = require_lib();
      function isSsh(input) {
        if (Array.isArray(input)) {
          return input.indexOf("ssh") !== -1 || input.indexOf("rsync") !== -1;
        }
        if (typeof input !== "string") {
          return false;
        }
        var prots = protocols(input);
        input = input.substring(input.indexOf("://") + 3);
        if (isSsh(prots)) {
          return true;
        }
        var urlPortPattern = new RegExp(".([a-zA-Z\\d]+):(\\d+)/");
        return !input.match(urlPortPattern) && input.indexOf("@") < input.indexOf(":");
      }
      module.exports = isSsh;
    }
  });

  // node_modules/git-up/lib/index.js
  var require_lib4 = __commonJS({
    "node_modules/git-up/lib/index.js"(exports, module) {
      "use strict";
      var parseUrl = require_dist();
      var isSsh = require_lib3();
      function gitUp(input) {
        var output = parseUrl(input);
        output.token = "";
        if (output.password === "x-oauth-basic") {
          output.token = output.user;
        } else if (output.user === "x-token-auth") {
          output.token = output.password;
        }
        if (isSsh(output.protocols) || output.protocols.length === 0 && isSsh(input)) {
          output.protocol = "ssh";
        } else if (output.protocols.length) {
          output.protocol = output.protocols[0];
        } else {
          output.protocol = "file";
          output.protocols = ["file"];
        }
        output.href = output.href.replace(/\/$/, "");
        return output;
      }
      module.exports = gitUp;
    }
  });

  // node_modules/git-url-parse/lib/index.js
  var require_lib5 = __commonJS({
    "node_modules/git-url-parse/lib/index.js"(exports, module) {
      "use strict";
      var gitUp = require_lib4();
      function gitUrlParse2(url) {
        if (typeof url !== "string") {
          throw new Error("The url must be a string.");
        }
        var shorthandRe = /^([a-z\d-]{1,39})\/([-\.\w]{1,100})$/i;
        if (shorthandRe.test(url)) {
          url = "https://github.com/" + url;
        }
        var urlInfo = gitUp(url), sourceParts = urlInfo.resource.split("."), splits = null;
        urlInfo.toString = function(type) {
          return gitUrlParse2.stringify(this, type);
        };
        urlInfo.source = sourceParts.length > 2 ? sourceParts.slice(1 - sourceParts.length).join(".") : urlInfo.source = urlInfo.resource;
        urlInfo.git_suffix = /\.git$/.test(urlInfo.pathname);
        urlInfo.name = decodeURIComponent((urlInfo.pathname || urlInfo.href).replace(/(^\/)|(\/$)/g, "").replace(/\.git$/, ""));
        urlInfo.owner = decodeURIComponent(urlInfo.user);
        switch (urlInfo.source) {
          case "git.cloudforge.com":
            urlInfo.owner = urlInfo.user;
            urlInfo.organization = sourceParts[0];
            urlInfo.source = "cloudforge.com";
            break;
          case "visualstudio.com":
            if (urlInfo.resource === "vs-ssh.visualstudio.com") {
              splits = urlInfo.name.split("/");
              if (splits.length === 4) {
                urlInfo.organization = splits[1];
                urlInfo.owner = splits[2];
                urlInfo.name = splits[3];
                urlInfo.full_name = splits[2] + "/" + splits[3];
              }
              break;
            } else {
              splits = urlInfo.name.split("/");
              if (splits.length === 2) {
                urlInfo.owner = splits[1];
                urlInfo.name = splits[1];
                urlInfo.full_name = "_git/" + urlInfo.name;
              } else if (splits.length === 3) {
                urlInfo.name = splits[2];
                if (splits[0] === "DefaultCollection") {
                  urlInfo.owner = splits[2];
                  urlInfo.organization = splits[0];
                  urlInfo.full_name = urlInfo.organization + "/_git/" + urlInfo.name;
                } else {
                  urlInfo.owner = splits[0];
                  urlInfo.full_name = urlInfo.owner + "/_git/" + urlInfo.name;
                }
              } else if (splits.length === 4) {
                urlInfo.organization = splits[0];
                urlInfo.owner = splits[1];
                urlInfo.name = splits[3];
                urlInfo.full_name = urlInfo.organization + "/" + urlInfo.owner + "/_git/" + urlInfo.name;
              }
              break;
            }
          // Azure DevOps (formerly Visual Studio Team Services)
          case "dev.azure.com":
          case "azure.com":
            if (urlInfo.resource === "ssh.dev.azure.com") {
              splits = urlInfo.name.split("/");
              if (splits.length === 4) {
                urlInfo.organization = splits[1];
                urlInfo.owner = splits[2];
                urlInfo.name = splits[3];
              }
              break;
            } else {
              splits = urlInfo.name.split("/");
              if (splits.length === 5) {
                urlInfo.organization = splits[0];
                urlInfo.owner = splits[1];
                urlInfo.name = splits[4];
                urlInfo.full_name = "_git/" + urlInfo.name;
              } else if (splits.length === 3) {
                urlInfo.name = splits[2];
                if (splits[0] === "DefaultCollection") {
                  urlInfo.owner = splits[2];
                  urlInfo.organization = splits[0];
                  urlInfo.full_name = urlInfo.organization + "/_git/" + urlInfo.name;
                } else {
                  urlInfo.owner = splits[0];
                  urlInfo.full_name = urlInfo.owner + "/_git/" + urlInfo.name;
                }
              } else if (splits.length === 4) {
                urlInfo.organization = splits[0];
                urlInfo.owner = splits[1];
                urlInfo.name = splits[3];
                urlInfo.full_name = urlInfo.organization + "/" + urlInfo.owner + "/_git/" + urlInfo.name;
              }
              if (urlInfo.query && urlInfo.query["path"]) {
                urlInfo.filepath = urlInfo.query["path"].replace(/^\/+/g, "");
              }
              if (urlInfo.query && urlInfo.query["version"]) {
                urlInfo.ref = urlInfo.query["version"].replace(/^GB/, "");
              }
              break;
            }
          default:
            splits = urlInfo.name.split("/");
            var nameIndex = splits.length - 1;
            if (splits.length >= 2) {
              var dashIndex = splits.indexOf("-", 2);
              var blobIndex = splits.indexOf("blob", 2);
              var treeIndex = splits.indexOf("tree", 2);
              var commitIndex = splits.indexOf("commit", 2);
              var srcIndex = splits.indexOf("src", 2);
              var rawIndex = splits.indexOf("raw", 2);
              var editIndex = splits.indexOf("edit", 2);
              nameIndex = dashIndex > 0 ? dashIndex - 1 : blobIndex > 0 ? blobIndex - 1 : treeIndex > 0 ? treeIndex - 1 : commitIndex > 0 ? commitIndex - 1 : srcIndex > 0 ? srcIndex - 1 : rawIndex > 0 ? rawIndex - 1 : editIndex > 0 ? editIndex - 1 : nameIndex;
              urlInfo.owner = splits.slice(0, nameIndex).join("/");
              urlInfo.name = splits[nameIndex];
              if (commitIndex) {
                urlInfo.commit = splits[nameIndex + 2];
              }
            }
            urlInfo.ref = "";
            urlInfo.filepathtype = "";
            urlInfo.filepath = "";
            var offsetNameIndex = splits.length > nameIndex && splits[nameIndex + 1] === "-" ? nameIndex + 1 : nameIndex;
            if (splits.length > offsetNameIndex + 2 && ["raw", "src", "blob", "tree", "edit"].indexOf(splits[offsetNameIndex + 1]) >= 0) {
              urlInfo.filepathtype = splits[offsetNameIndex + 1];
              urlInfo.ref = splits[offsetNameIndex + 2];
              if (splits.length > offsetNameIndex + 3) {
                urlInfo.filepath = splits.slice(offsetNameIndex + 3).join("/");
              }
            }
            urlInfo.organization = urlInfo.owner;
            break;
        }
        if (!urlInfo.full_name) {
          urlInfo.full_name = urlInfo.owner;
          if (urlInfo.name) {
            urlInfo.full_name && (urlInfo.full_name += "/");
            urlInfo.full_name += urlInfo.name;
          }
        }
        if (urlInfo.owner.startsWith("scm/")) {
          urlInfo.source = "bitbucket-server";
          urlInfo.owner = urlInfo.owner.replace("scm/", "");
          urlInfo.organization = urlInfo.owner;
          urlInfo.full_name = urlInfo.owner + "/" + urlInfo.name;
        }
        var bitbucket = /(projects|users)\/(.*?)\/repos\/(.*?)((\/.*$)|$)/;
        var matches = bitbucket.exec(urlInfo.pathname);
        if (matches != null) {
          urlInfo.source = "bitbucket-server";
          if (matches[1] === "users") {
            urlInfo.owner = "~" + matches[2];
          } else {
            urlInfo.owner = matches[2];
          }
          urlInfo.organization = urlInfo.owner;
          urlInfo.name = matches[3];
          splits = matches[4].split("/");
          if (splits.length > 1) {
            if (["raw", "browse"].indexOf(splits[1]) >= 0) {
              urlInfo.filepathtype = splits[1];
              if (splits.length > 2) {
                urlInfo.filepath = splits.slice(2).join("/");
              }
            } else if (splits[1] === "commits" && splits.length > 2) {
              urlInfo.commit = splits[2];
            }
          }
          urlInfo.full_name = urlInfo.owner + "/" + urlInfo.name;
          if (urlInfo.query.at) {
            urlInfo.ref = urlInfo.query.at;
          } else {
            urlInfo.ref = "";
          }
        }
        return urlInfo;
      }
      gitUrlParse2.stringify = function(obj, type) {
        type = type || (obj.protocols && obj.protocols.length ? obj.protocols.join("+") : obj.protocol);
        var port = obj.port ? ":" + obj.port : "";
        var user = obj.user || "git";
        var maybeGitSuffix = obj.git_suffix ? ".git" : "";
        switch (type) {
          case "ssh":
            if (port) return "ssh://" + user + "@" + obj.resource + port + "/" + obj.full_name + maybeGitSuffix;
            else return user + "@" + obj.resource + ":" + obj.full_name + maybeGitSuffix;
          case "git+ssh":
          case "ssh+git":
          case "ftp":
          case "ftps":
            return type + "://" + user + "@" + obj.resource + port + "/" + obj.full_name + maybeGitSuffix;
          case "http":
          case "https":
            var auth = obj.token ? buildToken(obj) : obj.user && (obj.protocols.includes("http") || obj.protocols.includes("https")) ? obj.user + "@" : "";
            return type + "://" + auth + obj.resource + port + "/" + buildPath(obj) + maybeGitSuffix;
          default:
            return obj.href;
        }
      };
      function buildToken(obj) {
        switch (obj.source) {
          case "bitbucket.org":
            return "x-token-auth:" + obj.token + "@";
          default:
            return obj.token + "@";
        }
      }
      function buildPath(obj) {
        switch (obj.source) {
          case "bitbucket-server":
            return "scm/" + obj.full_name;
          default:
            return "" + obj.full_name;
        }
      }
      module.exports = gitUrlParse2;
    }
  });

  // src/urls.js
  var import_git_url_parse = __toESM(require_lib5());
  var isSHA = (revision) => {
    if (revision.startsWith("sha256:")) {
      const hashOnly = revision.replace("sha256:", "");
      return hashOnly.match(/^[a-f0-9]{8,69}$/) !== null;
    }
    return revision.match(/^[a-f0-9]{5,40}$/) !== null;
  };
  function isValidURL(url) {
    try {
      const parsedUrl = new URL(url);
      return parsedUrl.protocol !== "javascript:" && parsedUrl.protocol !== "data:" && parsedUrl.protocol !== "vbscript:";
    } catch (e) {
      try {
        const parsedUrl = new URL(url, window.location.origin);
        return parsedUrl.protocol !== "javascript:" && parsedUrl.protocol !== "data:" && parsedUrl.protocol !== "vbscript:";
      } catch (e2) {
        return false;
      }
    }
  }
  function isBitbucketServer(parsed) {
    return parsed.source === "bitbucket-server" || parsed.resource.startsWith("bitbucket") && parsed.source !== "bitbucket.org";
  }
  function supportedSource(parsed) {
    return parsed.resource.startsWith("github") || parsed.source === "bitbucket.org" || isBitbucketServer(parsed) || parsed.source === "gitlab.com";
  }
  function bitbucketServerBrowseUrl(parsed) {
    const port = parsed.port ? `:${parsed.port}` : "";
    const host = `${protocol(parsed.protocol)}://${parsed.resource}${port}`;
    const owner = parsed.owner;
    if (owner.startsWith("~")) {
      return `${host}/users/${owner.slice(1)}/repos/${parsed.name}`;
    }
    return `${host}/projects/${owner}/repos/${parsed.name}`;
  }
  function protocol(proto) {
    return proto === "ssh" ? "https" : proto;
  }
  function repoUrl(url) {
    try {
      const parsed = (0, import_git_url_parse.default)(url);
      if (!supportedSource(parsed)) {
        return null;
      }
      if (isBitbucketServer(parsed)) {
        const browseUrl = bitbucketServerBrowseUrl(parsed);
        if (isValidURL(browseUrl)) {
          return browseUrl;
        }
        return null;
      }
      const parsedUrl = `${protocol(parsed.protocol)}://${parsed.resource}/${parsed.owner}/${parsed.name}`;
      if (!isValidURL(parsedUrl)) {
        return null;
      }
      return parsedUrl;
    } catch (e) {
      return null;
    }
  }
  function revisionUrl(url, revision, forPath) {
    let parsed;
    try {
      parsed = (0, import_git_url_parse.default)(url);
    } catch (e) {
      return null;
    }
    if (!supportedSource(parsed)) {
      return null;
    }
    if (isBitbucketServer(parsed)) {
      const base = bitbucketServerBrowseUrl(parsed);
      if (isSHA(revision) && !forPath) {
        return `${base}/commits/${revision}`;
      }
      if (!revision || revision === "HEAD") {
        return `${base}/browse`;
      }
      return `${base}/browse?at=${encodeURIComponent(revision)}`;
    }
    let urlSubPath = isSHA(revision) ? "commit" : "tree";
    if (parsed.source === "bitbucket.org") {
      urlSubPath = isSHA(revision) && !forPath ? "commits" : "src";
    }
    if (parsed.source === "gitlab.com") {
      urlSubPath = "-/" + urlSubPath;
    }
    return `${protocol(parsed.protocol)}://${parsed.resource}/${parsed.owner}/${parsed.name}/${urlSubPath}/${revision || "HEAD"}`;
  }

  // src/extension-monorepo-controller.js
  ((window2) => {
    const extensionsAPI = window2.extensionsAPI;
    if (!extensionsAPI || !extensionsAPI.registerStatusPanelExtension) {
      console.error("argocd-monorepo-controller: extensionsAPI is not available, extension not registered");
      return;
    }
    const React = window2.React;
    if (!React) {
      console.error("argocd-monorepo-controller: React is not available, extension not registered");
      return;
    }
    const ReactDOM = window2.ReactDOM;
    const canRenderPopup = !!(ReactDOM && ReactDOM.createPortal && React.useState && React.useRef && window2.document && window2.document.body);
    const TITLE = "Change Revision";
    const ID = "monorepo_change_revision";
    const CHANGE_REVISION_ANN = "mrp-controller.argoproj.io/change-revision";
    const CHANGE_REVISIONS_ANN = "mrp-controller.argoproj.io/change-revisions";
    const UNKNOWN_REVISION = "\u2014";
    const UNKNOWN_REVISION_TITLE = "No change revision has been calculated for this source yet";
    const HELP_TEXT = "The commit that actually changed the manifests generated by this application, as opposed to the most recent commit in the repository. Multi source applications show one revision per source, and Helm repository sources show a chart version. Maintained by the Argo CD Monorepo Controller.";
    const labelStyle = {
      display: "flex",
      alignItems: "flex-start",
      fontSize: "12px",
      fontWeight: 600,
      color: "#6d7f8b",
      minHeight: "18px"
    };
    const rowsStyle = {
      display: "grid",
      gridTemplateColumns: "auto 1fr",
      columnGap: "0.5em",
      alignItems: "baseline"
    };
    const sourceNameStyle = {
      color: "#6d7f8b",
      maxWidth: "10em",
      overflow: "hidden",
      textOverflow: "ellipsis",
      whiteSpace: "nowrap"
    };
    const revisionStyle = { fontFamily: "monospace" };
    const TOOLTIP_MAX_WIDTH = 350;
    const TOOLTIP_GAP = 8;
    const tooltipPopup = (content, anchor) => {
      const below = anchor.top < 80;
      const placement = below ? "bottom" : "top";
      return React.createElement(
        "div",
        {
          className: "tippy-popper",
          "data-placement": placement,
          style: {
            position: "fixed",
            left: Math.round(anchor.left + anchor.width / 2),
            top: Math.round(below ? anchor.bottom + TOOLTIP_GAP : anchor.top - TOOLTIP_GAP),
            transform: below ? "translateX(-50%)" : "translate(-50%, -100%)",
            zIndex: 1e3
          }
        },
        React.createElement(
          "div",
          {
            className: "tippy-tooltip light-theme",
            "data-placement": placement,
            "data-state": "visible",
            style: { maxWidth: TOOLTIP_MAX_WIDTH + "px", textAlign: "left" }
          },
          // the stylesheet only gives the arrow its shape and colour, tippy
          // places it with inline styles
          React.createElement("div", { className: "tippy-arrow", style: { left: "50%", marginLeft: "-8px" } }),
          React.createElement("div", { className: "tippy-content" }, content)
        )
      );
    };
    const PopupTooltip = (props) => {
      const [anchor, setAnchor] = React.useState(null);
      const trigger = React.useRef(null);
      const hide = () => setAnchor(null);
      const show = () => {
        const node = trigger.current;
        if (!node || !node.getBoundingClientRect) {
          return;
        }
        const rect = node.getBoundingClientRect();
        setAnchor({ left: rect.left, top: rect.top, bottom: rect.bottom, width: rect.width });
      };
      React.useEffect(() => {
        if (!anchor) {
          return void 0;
        }
        window2.addEventListener("scroll", hide, true);
        window2.addEventListener("resize", hide);
        return () => {
          window2.removeEventListener("scroll", hide, true);
          window2.removeEventListener("resize", hide);
        };
      }, [anchor]);
      const target = React.cloneElement(props.children, { ref: trigger, onMouseEnter: show, onMouseLeave: hide });
      if (!anchor) {
        return target;
      }
      return React.createElement(
        React.Fragment,
        null,
        target,
        ReactDOM.createPortal(tooltipPopup(props.content, anchor), window2.document.body)
      );
    };
    const TitleTooltip = (props) => React.cloneElement(props.children, { title: props.text });
    const Tooltip = canRenderPopup ? PopupTooltip : TitleTooltip;
    const helpIcon = (text) => React.createElement(
      Tooltip,
      { content: text, text },
      React.createElement(
        "span",
        { style: { marginLeft: "5px", cursor: "help" } },
        React.createElement(
          "span",
          { style: { fontSize: "smaller" } },
          " ",
          React.createElement("i", { className: "fa fa-question-circle help-tip" })
        )
      )
    );
    const getSources = (application) => {
      const spec = application && application.spec || {};
      if (Array.isArray(spec.sources) && spec.sources.length > 0) {
        return spec.sources;
      }
      if (spec.source) {
        return [spec.source];
      }
      return [];
    };
    const getChangeRevisions = (application) => {
      const annotations = application && application.metadata && application.metadata.annotations || {};
      const revisions = annotations[CHANGE_REVISIONS_ANN];
      if (typeof revisions === "string") {
        let parsed;
        try {
          parsed = JSON.parse(revisions);
        } catch (e) {
          parsed = null;
        }
        if (Array.isArray(parsed)) {
          return parsed.map((revision2) => typeof revision2 === "string" ? revision2 : "");
        }
      }
      const revision = annotations[CHANGE_REVISION_ANN];
      if (typeof revision === "string") {
        return [revision];
      }
      return null;
    };
    const isFullSHA = (revision) => /^[0-9a-f]{40}$/.test(revision);
    const formatRevision = (revision, source, withChartName) => {
      if (!revision) {
        return UNKNOWN_REVISION;
      }
      if (source && source.chart) {
        return withChartName ? source.chart + ":" + revision : revision;
      }
      if (isFullSHA(revision)) {
        return revision.substring(0, 7);
      }
      return revision;
    };
    const commitURL = (revision, source) => {
      if (!revision || !source || !source.repoURL || source.chart) {
        return null;
      }
      return revisionUrl(source.repoURL, revision, false);
    };
    const maybeLink = (url, content) => {
      if (!url) {
        return content;
      }
      return React.createElement(
        "a",
        { href: url, target: "_blank", rel: "noopener noreferrer" },
        content,
        " ",
        React.createElement("i", { className: "fa fa-external-link-alt" })
      );
    };
    const sourceName = (source, index) => {
      if (source && source.name) {
        return source.name;
      }
      if (source && source.chart) {
        return source.chart;
      }
      const repoURL = source && source.repoURL;
      if (repoURL) {
        const trimmed = repoURL.replace(/\/+$/, "").replace(/\.git$/, "");
        let name = trimmed.substring(trimmed.lastIndexOf("/") + 1);
        if (name.indexOf(":") >= 0) {
          name = name.substring(name.lastIndexOf(":") + 1);
        }
        if (name) {
          return name;
        }
      }
      return "source " + index;
    };
    const revisionElement = (revision, source, withChartName, key) => React.createElement(
      Tooltip,
      {
        key,
        content: revision ? React.createElement("span", { style: revisionStyle }, revision) : UNKNOWN_REVISION_TITLE,
        text: revision || UNKNOWN_REVISION_TITLE
      },
      React.createElement(
        "div",
        { style: revisionStyle },
        maybeLink(commitURL(revision, source), formatRevision(revision, source, withChartName))
      )
    );
    const sourceNameElement = (source, index) => React.createElement(
      "div",
      { key: "name-" + index, style: sourceNameStyle, title: source && source.repoURL || void 0 },
      sourceName(source, index)
    );
    const revisionMetadataURL = (application, revision, sourceIndex) => {
      const metadata = application.metadata || {};
      const url = new window2.URL(
        "api/v1/applications/" + encodeURIComponent(metadata.name) + "/revisions/" + encodeURIComponent(revision) + "/metadata",
        window2.document.baseURI
      );
      if (metadata.namespace) {
        url.searchParams.set("appNamespace", metadata.namespace);
      }
      if (application.spec && application.spec.project) {
        url.searchParams.set("project", application.spec.project);
      }
      url.searchParams.set("sourceIndex", String(sourceIndex));
      return url.toString();
    };
    const errorText = (body, status) => {
      if (body && typeof body === "object") {
        if (typeof body.error === "string" && body.error) {
          return body.error;
        }
        if (typeof body.message === "string" && body.message) {
          return body.message;
        }
      }
      return "request failed with status " + status;
    };
    const fetchRevisionMetadata = (application, revision, sourceIndex) => window2.fetch(revisionMetadataURL(application, revision, sourceIndex), {
      credentials: "same-origin",
      headers: { Accept: "application/json" }
    }).then(
      (response) => response.json().catch(() => null).then((body) => {
        if (!response.ok) {
          throw new Error(errorText(body, response.status));
        }
        return body || {};
      })
    );
    const detailRow = (label, value, key, valueStyle) => React.createElement(
      "div",
      { className: "row white-box__details-row", key },
      React.createElement("div", { className: "columns small-3" }, label),
      React.createElement("div", { className: "columns small-9", style: valueStyle }, value)
    );
    const formatDate = (value) => {
      const date = new Date(value);
      return isNaN(date.getTime()) ? value : date.toLocaleString();
    };
    const metadataRows = (result) => {
      if (!result) {
        return [];
      }
      if (result.state === "loading") {
        return [detailRow("Commit", "Loading commit details...", "loading")];
      }
      if (result.state === "error") {
        return [detailRow("Commit", result.error, "error", { color: "#e96d76" })];
      }
      const metadata = result.data || {};
      const rows = [];
      if (metadata.author) {
        const author = metadata.signatureInfo ? metadata.author + " - " + metadata.signatureInfo : metadata.author;
        rows.push(detailRow("Author", author, "author"));
      }
      if (metadata.date) {
        rows.push(detailRow("Date", formatDate(metadata.date), "date"));
      }
      if (metadata.tags && metadata.tags.length) {
        rows.push(detailRow("Tags", metadata.tags.join(", "), "tags"));
      }
      if (metadata.message) {
        rows.push(detailRow("Message", metadata.message, "message", { whiteSpace: "pre-wrap" }));
      }
      if (rows.length === 0) {
        rows.push(detailRow("Commit", "No commit details available", "empty"));
      }
      return rows;
    };
    const sourceBox = (source, revision, result, index) => {
      const rows = [];
      if (source && source.repoURL) {
        rows.push(detailRow("Repository", maybeLink(repoUrl(source.repoURL), source.repoURL), "repo"));
      }
      if (source && source.chart) {
        rows.push(detailRow("Chart", source.chart, "chart"));
      } else if (source && source.path) {
        rows.push(detailRow("Path", source.path, "path"));
      }
      if (source && source.targetRevision) {
        rows.push(detailRow("Target revision", source.targetRevision, "target"));
      }
      rows.push(
        detailRow(
          "Change revision",
          revision ? maybeLink(commitURL(revision, source), revision) : UNKNOWN_REVISION,
          "revision",
          revision ? revisionStyle : void 0
        )
      );
      metadataRows(result).forEach((row) => rows.push(row));
      return React.createElement(
        "div",
        { className: "white-box", key: "source-" + index, style: { marginBottom: "1em" } },
        React.createElement("p", null, sourceName(source, index)),
        React.createElement("div", { className: "white-box__details" }, rows)
      );
    };
    const MIDDLE_CLASS = "sliding-panel--is-middle";
    const CLOSE_ANIMATION_MS = 500;
    const useMiddlePanelWidth = (ref) => {
      React.useEffect(() => {
        const node = ref.current;
        const panel = node && node.closest && node.closest(".sliding-panel");
        if (!panel || !panel.classList || panel.classList.contains(MIDDLE_CLASS)) {
          return void 0;
        }
        panel.classList.add(MIDDLE_CLASS);
        return () => {
          window2.setTimeout(() => panel.classList.remove(MIDDLE_CLASS), CLOSE_ANIMATION_MS);
        };
      }, []);
    };
    const ChangeRevisionFlyout = (props) => {
      const application = props && props.application;
      const [results, setResults] = React.useState({});
      const rootRef = React.useRef(null);
      useMiddlePanelWidth(rootRef);
      const revisions = getChangeRevisions(application) || [];
      const sources = getSources(application);
      const rowCount = sources.length > 0 ? Math.min(revisions.length, sources.length) : revisions.length;
      React.useEffect(() => {
        let cancelled = false;
        const pending = {};
        for (let i = 0; i < rowCount; i++) {
          const source = sources[i];
          if (!revisions[i] || source && source.chart) {
            continue;
          }
          pending[i] = { state: "loading" };
        }
        if (Object.keys(pending).length === 0) {
          return void 0;
        }
        setResults(pending);
        Object.keys(pending).forEach((key) => {
          const index = Number(key);
          fetchRevisionMetadata(application, revisions[index], index).then(
            (data) => {
              if (!cancelled) {
                setResults((current) => Object.assign({}, current, { [index]: { state: "ok", data } }));
              }
            },
            (error) => {
              if (!cancelled) {
                setResults(
                  (current) => Object.assign({}, current, { [index]: { state: "error", error: String(error.message || error) } })
                );
              }
            }
          );
        });
        return () => {
          cancelled = true;
        };
      }, [application && application.metadata && application.metadata.name, revisions.join(",")]);
      if (rowCount === 0) {
        return React.createElement(
          "div",
          { ref: rootRef },
          React.createElement("h4", null, TITLE),
          React.createElement("p", null, "This application has no change revision.")
        );
      }
      const boxes = [];
      for (let i = 0; i < rowCount; i++) {
        boxes.push(sourceBox(sources[i], revisions[i], results[i], i));
      }
      return React.createElement(
        "div",
        { ref: rootRef },
        React.createElement("h4", null, TITLE),
        React.createElement("p", null, "The commit that last changed the manifests generated by this application."),
        boxes
      );
    };
    const ChangeRevisionPanelItem = (props) => {
      const application = props && props.application;
      const revisions = getChangeRevisions(application);
      if (!revisions || revisions.length === 0) {
        return null;
      }
      const sources = getSources(application);
      const rowCount = sources.length > 0 ? Math.min(revisions.length, sources.length) : revisions.length;
      if (rowCount === 0) {
        return null;
      }
      let value;
      if (rowCount === 1) {
        value = revisionElement(revisions[0], sources[0], true);
      } else {
        const rows = [];
        for (let i = 0; i < rowCount; i++) {
          rows.push(sourceNameElement(sources[i], i));
          rows.push(revisionElement(revisions[i], sources[i], false, "revision-" + i));
        }
        value = React.createElement("div", { style: rowsStyle }, rows);
      }
      return React.createElement(
        "div",
        { className: "application-status-panel__item" },
        // sectionHeader() in application-status-panel.tsx lays the built in
        // items out this way: the label, then the button that opens the flyout
        React.createElement(
          "div",
          { style: { display: "flex", alignItems: "center" } },
          React.createElement(
            "label",
            { style: labelStyle },
            rowCount > 1 ? "CHANGE REVISIONS" : "CHANGE REVISION",
            helpIcon(HELP_TEXT)
          ),
          React.createElement(
            "button",
            {
              className: "argo-button application-status-panel__more-button",
              onClick: () => props.openFlyout && props.openFlyout()
            },
            React.createElement("i", { className: "fa fa-ellipsis-h" })
          )
        ),
        React.createElement("div", { className: "application-status-panel__item-value" }, value)
      );
    };
    extensionsAPI.registerStatusPanelExtension(ChangeRevisionPanelItem, TITLE, ID, ChangeRevisionFlyout);
  })(window);
})();
/*! Bundled license information:

git-url-parse/lib/index.js:
  (*!
   * buildToken
   * Builds OAuth token prefix (helper function)
   *
   * @name buildToken
   * @function
   * @param {GitUrl} obj The parsed Git url object.
   * @return {String} token prefix
   *)
*/
