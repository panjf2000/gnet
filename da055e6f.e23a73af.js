(window.webpackJsonp=window.webpackJsonp||[]).push([[87],{231:function(e,t,a){"use strict";a.r(t),a.d(t,"frontMatter",(function(){return r})),a.d(t,"metadata",(function(){return i})),a.d(t,"rightToc",(function(){return f})),a.d(t,"default",(function(){return o}));var c=a(1),b=a(9),n=(a(0),a(251)),r={last_modified_on:"2021-12-05",$schema:"/.meta/.schemas/highlights.json",title:"Released gnet v1.6.0",description:"Released the official stable version of v1.6.0",author_github:"https://github.com/panjf2000",pr_numbers:["b8d571d"],release:"1.6.0",hide_on_release_notes:!1,tags:["type: tag","domain: v1.6.0"]},i={date:"2021-12-05T00:00:00.000Z",description:"Released the official stable version of v1.6.0",permalink:"/highlights/2021-12-05-released-1-6-0",readingTime:"2 min read",source:"@site/highlights/2021-12-05-released-1-6-0.md",tags:[{label:"type: tag",permalink:"/highlights/tags/type-tag"},{label:"domain: v1.6.0",permalink:"/highlights/tags/domain-v-1-6-0"}],title:"Released gnet v1.6.0",truncated:!1,prevItem:{title:"Release of gnet v2.0.0",permalink:"/highlights/2022-02-27-release-of-gnet-v2"},nextItem:{title:"Released gnet v1.5.2",permalink:"/highlights/2021-07-20-released-1-5-2"}},f=[{value:"Features",id:"features",children:[]},{value:"Enhancements",id:"enhancements",children:[]},{value:"Bugfixes",id:"bugfixes",children:[]},{value:"Docs",id:"docs",children:[]},{value:"Misc",id:"misc",children:[]}],m={rightToc:f};function o(e){var t=e.components,a=Object(b.a)(e,["components"]);return Object(n.b)("wrapper",Object(c.a)({},m,a,{components:t,mdxType:"MDXLayout"}),Object(n.b)("h1",{id:"gnet-client-is-now-available-for-production"},"gnet client is now available for production!"),Object(n.b)("h2",{id:"features"},"Features"),Object(n.b)("ul",null,Object(n.b)("li",{parentName:"ul"},"Add a new event handler: AfterWrite() ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/6a654c85e7c1172503813c9703603e42eea2fc29"}),"6a654c")),Object(n.b)("li",{parentName:"ul"},"Implement the gnet client ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/2295e8c6f3394341d28318cb6ea33f0799d52c45"}),"2295e8")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/a5ac95a5057fb82e2f71cb6a7f4ffed83c967efb"}),"a5ac95")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/4db46da43d5defd5da71213c0abaebb174af642c"}),"4db46d")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/802fa358f2c8ac95414e36cb0afd53f6dd57bfa0"}),"802fa3")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/7159b95cd9ebc8fe2f9bea909844eb8c8bb37bf7"}),"7159b9")),Object(n.b)("li",{parentName:"ul"},"Implement writev and readv on BSD-like OS's ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/60ba6d30b04351e26c3f7c9cc496b1b849936731"}),"60ba6d")),Object(n.b)("li",{parentName:"ul"},"Implement a mixed buffer of ring-buffer and list-buffer ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/edbdf4b54b7439bfb2ac4ba9652ec6a1764e0659"}),"edbdf4")),Object(n.b)("li",{parentName:"ul"},"Invoke OnClosed() when a UDP socket is closed ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/7be4b2a758e32af489450b6b62d8da48e471ba00"}),"7be4b2")),Object(n.b)("li",{parentName:"ul"},"Implement the gnet.Conn.AsyncWritev() ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/9a2032f876cd8f41c554545bcbb63d3043f4946f"}),"9a2032"))),Object(n.b)("h2",{id:"enhancements"},"Enhancements"),Object(n.b)("ul",null,Object(n.b)("li",{parentName:"ul"},"Prevent the event-list from expanding or shrinking endlessly ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/b220dfd3f3ff9b8ecee4a09170d4db3760393fc0"}),"b220df")),Object(n.b)("li",{parentName:"ul"},"Reduce the potential system calls for waking pollers up ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/9ce41f3b921a9341081506629185e733f97defa4"}),"9ce41f")),Object(n.b)("li",{parentName:"ul"},"Eliminate the code for preventing false-sharing ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/0bfade3aea015a7932b0e45b646a6c85a620a205"}),"0bfade")),Object(n.b)("li",{parentName:"ul"},"Support so_reuseaddr ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/pull/280"}),"#280")),Object(n.b)("li",{parentName:"ul"},"Make several improvements for logger ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/58d2031440b1c9725e2d12aeb651aa8bc78d3489"}),"58d203")),Object(n.b)("li",{parentName:"ul"},"Optimize the buffer management and network I/O ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/6aba6d7a3fc31cf749b0001dcb1c82f01c816f65"}),"6aba6d")),Object(n.b)("li",{parentName:"ul"},"Improve the project layout ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/2e172bde78bcdb56dbec9a57d95dfa4b6213b1f2"}),"2e172b")),Object(n.b)("li",{parentName:"ul"},"Improve the logic of reading data from socket into ring-buffer ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/a7f07b3d4eaa70a9b5c8b389d73b72ddb06b8c16"}),"a7f07b")),Object(n.b)("li",{parentName:"ul"},"Get as much data read from socket per loop as possible ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/148ee163fb3ddd0fcd7919ab17390a3cd910933f"}),"148ee1")),Object(n.b)("li",{parentName:"ul"},"Improve the network read with ring-buffer and readv ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/0dcf599fd0673bc712b5409fd9a0711cb90606c0"}),"0dcf59")),Object(n.b)("li",{parentName:"ul"},"Avoid memory allocations when calling readv ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/15611b482f50f1333fcee47b02d6ec04b4d2ede5"}),"15611b")),Object(n.b)("li",{parentName:"ul"},"Refactor the logic of handling UDP sockets ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/d72d3de70a0cb31c6059820dbd4ba6db6c4e23eb"}),"d72d3d")),Object(n.b)("li",{parentName:"ul"},"Make the mixed-buffer more flexible ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/4ac906cae698b1a4483c583d0267f86f05ce595b"}),"d72d3d")),Object(n.b)("li",{parentName:"ul"},"Improve the management logic of the mixed-buffer ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/b8d571dd762cb79c2c685f16d36886f6edb40195"}),"b8d571"))),Object(n.b)("h2",{id:"bugfixes"},"Bugfixes"),Object(n.b)("ul",null,Object(n.b)("li",{parentName:"ul"},"Resolve the data race of stdConn on Windows ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/pull/235"}),"#235")),Object(n.b)("li",{parentName:"ul"},"Fix the data corruption in some default codecs ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/a56d2f3f50981107ae6b2bd2653fe19dc75d4e18"}),"a56d2f")),Object(n.b)("li",{parentName:"ul"},"Fix the issue of panic: runtime error: slice bounds out of range ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/30311e936869d8685c8c06ff98170f0adb68bc8b"}),"30311e"))),Object(n.b)("h2",{id:"docs"},"Docs"),Object(n.b)("ul",null,Object(n.b)("li",{parentName:"ul"},"Update the benchmark data ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/21f55a6832d82b88073c51ccfbed8a0e627399c3"}),"21f55a")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/24e4ce06a4c4e1d3990eec9945c98175763c027f"}),"24e4ce")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/1b4ae56edf45bb3bc165c183a089fb0a8144ca67"}),"1b4ae5")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/81d984236401fb42d2f75c8989b87321804f4503"}),"81d984")),Object(n.b)("li",{parentName:"ul"},"Add the echo benchmarks on macOS ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/f429e7afaf3745574c95bf03d60baeaec2ecd9c1"}),"f429e7")),Object(n.b)("li",{parentName:"ul"},"Change the license from MIT to Apache 2.0 ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/a900c8f21958eb8096443125afafb672d9f1218e"}),"a900c8"))),Object(n.b)("h2",{id:"misc"},"Misc"),Object(n.b)("ul",null,Object(n.b)("li",{parentName:"ul"},"Add a new patron ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/0c9f965f24a6a706ddcfbcc2ba2dd8339e611e8e"}),"0c9f96")),Object(n.b)("li",{parentName:"ul"},"Create FUNDING.yml ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/1989eda4cc668e548f8572ac9fb07cef8c8f612d"}),"1989ed")," ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/7b29795db5fe184da0939490f8bf4ec39d3c27db"}),"7b2979")),Object(n.b)("li",{parentName:"ul"},"Remove the irrelevant articles ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/bbdc1bcc76138feb3529d639e63ebe9374c22165"}),"bbdc1b")),Object(n.b)("li",{parentName:"ul"},"Correct the wrong logging function ",Object(n.b)("a",Object(c.a)({parentName:"li"},{href:"https://github.com/panjf2000/gnet/commit/10c619f3a42c4f8397464a7a45daff24bfa873ea"}),"10c619"))))}o.isMDXComponent=!0},251:function(e,t,a){"use strict";a.d(t,"a",(function(){return l})),a.d(t,"b",(function(){return h}));var c=a(0),b=a.n(c);function n(e,t,a){return t in e?Object.defineProperty(e,t,{value:a,enumerable:!0,configurable:!0,writable:!0}):e[t]=a,e}function r(e,t){var a=Object.keys(e);if(Object.getOwnPropertySymbols){var c=Object.getOwnPropertySymbols(e);t&&(c=c.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),a.push.apply(a,c)}return a}function i(e){for(var t=1;t<arguments.length;t++){var a=null!=arguments[t]?arguments[t]:{};t%2?r(Object(a),!0).forEach((function(t){n(e,t,a[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(a)):r(Object(a)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(a,t))}))}return e}function f(e,t){if(null==e)return{};var a,c,b=function(e,t){if(null==e)return{};var a,c,b={},n=Object.keys(e);for(c=0;c<n.length;c++)a=n[c],t.indexOf(a)>=0||(b[a]=e[a]);return b}(e,t);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);for(c=0;c<n.length;c++)a=n[c],t.indexOf(a)>=0||Object.prototype.propertyIsEnumerable.call(e,a)&&(b[a]=e[a])}return b}var m=b.a.createContext({}),o=function(e){var t=b.a.useContext(m),a=t;return e&&(a="function"==typeof e?e(t):i({},t,{},e)),a},l=function(e){var t=o(e.components);return b.a.createElement(m.Provider,{value:t},e.children)},p={inlineCode:"code",wrapper:function(e){var t=e.children;return b.a.createElement(b.a.Fragment,{},t)}},d=Object(c.forwardRef)((function(e,t){var a=e.components,c=e.mdxType,n=e.originalType,r=e.parentName,m=f(e,["components","mdxType","originalType","parentName"]),l=o(a),d=c,h=l["".concat(r,".").concat(d)]||l[d]||p[d]||n;return a?b.a.createElement(h,i({ref:t},m,{components:a})):b.a.createElement(h,i({ref:t},m))}));function h(e,t){var a=arguments,c=t&&t.mdxType;if("string"==typeof e||c){var n=a.length,r=new Array(n);r[0]=d;var i={};for(var f in t)hasOwnProperty.call(t,f)&&(i[f]=t[f]);i.originalType=e,i.mdxType="string"==typeof e?e:c,r[1]=i;for(var m=2;m<n;m++)r[m]=a[m];return b.a.createElement.apply(null,r)}return b.a.createElement.apply(null,a)}d.displayName="MDXCreateElement"}}]);