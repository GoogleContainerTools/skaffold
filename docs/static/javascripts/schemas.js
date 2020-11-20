const schema_list = [...document.getElementById("schema_list").getElementsByTagName("li")].map(l => l.textContent);
const majors = {};
for (let s of schema_list) {
    let i = 2;
    for (; i < s.length; i++) {
        if (!s[i].match(/^[a-z]+$/)) {
            break;
        }
    }
    let minor = parseInt(s.substr(i));
    let major = s.substr(0, i);

    if (isNaN(minor)) {
        minor = "";
    }
    if (!(major in majors)) {
        majors[major] = [];
    }
    majors[major].push(minor);
}

Object.keys(majors)
    .sort()
    .forEach(function(major) {
        majors[major].sort((a,b) => a-b);
        for (const minor of majors[major]) {
            const version = major + minor;
            $("#schema_links").last().after($(`<a href="/docs/references/yaml/?version=${version}">${version}</a>`));
        }
    });
