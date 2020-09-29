import React from 'react';

import Heading from '@theme/Heading';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';

import styles from './community.module.css';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

const AnchoredH2 = Heading('h2');
const AnchoredH3 = Heading('h3');

function Community() {
  const context = useDocusaurusContext();
  const {siteConfig = {}} = context;
  const {metadata: {team}} = siteConfig.customFields;

  return (
    <Layout title="Community" description="Join the Gnet community. Connect with other Gnet users and help make Gnet better.">
      <header className="hero hero--clean">
        <div className="container container--fluid">
          <h1>Gnet Community</h1>
          <div className="hero--subtitle">Join the Gnet community. Connect with other Gnet users and help make Gnet better.</div>
        </div>
      </header>
      <main>
        <section>
          <div className="container">
            <div className="row">
              <div className="col">
                <a href="https://gitter.im/gnet-io/gnet" target="_blank" className="panel panel--button">
                  <div className="panel--icon">
                    <i className="feather icon-message-circle"></i>
                  </div>
                  <div className="panel--title">Chat</div>
                  <div className="panel--description">Ask questions and get help</div>
                </a>
              </div>
              <div className="col">
                <a href="https://twitter.com/_andy_pan" target="_blank" className="panel panel--button">
                  <div className="panel--icon">
                    <i className="feather icon-twitter" title="Twitter"></i>
                  </div>
                  <div className="panel--title">@_andy_pan</div>
                  <div className="panel--description">Follow me in real-time</div>
                </a>
              </div>
              <div className="col">
                <a href="https://github.com/panjf2000/gnet" target="_blank" className="panel panel--button">
                  <div className="panel--icon">
                    <i className="feather icon-github"></i>
                  </div>
                  <div className="panel--title">Github panjf2000/gnet</div>
                  <div className="panel--description">Code, issues, and pull requests</div>
                </a>
              </div>
            </div>
          </div>
        </section>
        <section>
          <div className="container">
            <AnchoredH2 id="team">Meet The Team</AnchoredH2>
            <div className="sub-title">Andy Pan is the creator of gnet and the only core contributor at present, hoping more developers will join me in the future.</div>

            <div className={styles.coreTeam}>
              {team.map((member, idx) => (
                <Link key={idx} to={member.github} className="avatar avatar--vertical">
                  <img
                    className="avatar__photo avatar__photo--xl"
                    src={member.avatar}
                  />
                  <div className="avatar__intro">
                    <h4 className="avatar__name">{member.name}</h4>
                  </div>
                </Link>
              ))}
            </div>
          </div>
        </section>
        <section>
          <div className="container">
            <AnchoredH2 id="faqs">FAQs</AnchoredH2>

            <AnchoredH3 id="contribute" className="header--flush">How do I contribute to Gnet?</AnchoredH3>

            <p>
              Gnet is <a href="https://github.com/panjf2000/gnet">open-source</a> and welcomes contributions. A few guidelines to help you get started:
            </p>
            <ol>
              <li>Read our <a href="https://github.com/panjf2000/gnet/blob/master/CONTRIBUTING.md" target="_blank">contribution guide</a>.</li>
              <li>Start with <a href="https://github.com/panjf2000/gnet/issues?q=is%3Aissue+label%3A%22good+first+issue%22" target="_blank">good first issues</a>.</li>
              <li>Join our <a href="https://gitter.im/gnet-io/gnet" target="_blank">chat</a> if you have any questions. We are happy to help!</li>
            </ol>

            <AnchoredH3 id="contribute" className="header--flush margin-top--lg">Why is Gnet so fast?</AnchoredH3>

            <p>
              Gnet's networking model is designed and tuned to handling millions of network connections/requests, which backs gnet up to be the fatest networking framework in Go.
              In addition to the first-class networking model, the implementation of auto-scaling and reusable ring-buffers in gnet is also one of the critical essentials for its high performance.
            </p>
            <ol>
              <li><a href="https://gnet.host/blog/presenting-gnet/#networking-model-of-multiple-threadsgoroutines">Networking models inside gnet</a></li>
              <li><a href="https://gnet.host/blog/presenting-gnet/#reusable-and-auto-scaling-ring-buffer">Reusable and auto-scaling ring-buffers in gnet</a></li>
              <li><a href="https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb/" target="_blank">A Million WebSockets and Go</a></li>
              <li><a href="https://speakerdeck.com/eranyanay/going-infinite-handling-1m-websockets-connections-in-go" target="_blank">Going Infinite, handling 1M websockets connections in Go</a></li>
            </ol>

            <AnchoredH3 id="contribute" className="header--flush margin-top--lg">How can I build networking applications of diverse protocols on top of gnet?</AnchoredH3>

            <p>
              There are some examples powered by gnet framework, go check out those source code and get an initial perception about developing networking applications based on gnei,
              after that, you can read the documentaions of gnet to learn all its APIs and try to wirte a demo application.
            </p>
            <ol>
              <li><a href="https://github.com/gnet-io/gnet-examples" target="_blank">Gnet Examples</a></li>
              <li><a href="https://pkg.go.dev/github.com/panjf2000/gnet?tab=doc" target="_blank">Gnet Documentaions</a></li>
            </ol>
          </div>
        </section>
      </main>
    </Layout>
  );
}

export default Community;
