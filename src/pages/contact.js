import React from 'react';

import Layout from '@theme/Layout';

import styles from './community.module.css';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

function Contact() {
  const context = useDocusaurusContext();
  const {siteConfig = {}} = context;
  const {metadata: {team}} = siteConfig.customFields;

  return (
    <Layout title="Contact" description="Contact the gnet author">
      <header className="hero">
        <div className="container container--fluid">
          <h1>Contact</h1>
          <div className="hero--subtitle">
            gnet is an open-source software created by <a href="https://github.com/panjf2000">Andy Pan</a>. You can contact the author using any of the options below.
          </div>
        </div>
      </header>
      <main>
        <section>
          <div className="container">
            <div className="row">
              <div className="col">
                <a href="mailto:i@andypan.me" className="panel text--center">
                  <div className="panel--icon">
                    <i className="feather icon-mail"></i>
                  </div>
                  <div className="panel--title">i@andypan.me</div>
                  <div className="panel--description">Send me an email</div>
                </a>
              </div>
              <div className="col">
                <a href="https://twitter.com/panjf2000" target="_blank" className="panel text--center">
                  <div className="panel--icon">
                    <i className="feather icon-twitter"></i>
                  </div>
                  <div className="panel--title">@panjf2000</div>
                  <div className="panel--description">
                    Tweet at me
                  </div>
                </a>
              </div>
              <div className="col">
                <a href="https://discord.gg/UyKD7NZcfH" target="_blank" className="panel text--center">
                  <div className="panel--icon">
                    <i className="feather icon-message-circle"></i>
                  </div>
                  <div className="panel--title">Chat</div>
                  <div className="panel--description">Join our chat</div>
                </a>
              </div>
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}

export default Contact;
