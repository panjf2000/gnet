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
    <Layout title="Community" description="Join the gnet community. Connect with other gnet users and help make gnet better.">
      <header className="hero hero--clean">
        <div className="container container--fluid">
          <h1>gnet Community</h1>
          <div className="hero--subtitle">Join the gnet community. Connect with other gnet users and help make gnet better.</div>
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
                <a href="https://twitter.com/panjf2000" target="_blank" className="panel panel--button">
                  <div className="panel--icon">
                    <i className="feather icon-twitter" title="Twitter"></i>
                  </div>
                  <div className="panel--title">@panjf2000</div>
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
      </main>
    </Layout>
  );
}

export default Community;
