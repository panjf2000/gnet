import React from 'react';

import Layout from '@theme/Layout';

import styles from './community.module.css';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

function Contact() {
  const context = useDocusaurusContext();
  const {siteConfig = {}} = context;
  const {metadata: {team}} = siteConfig.customFields;

  return (
    <Layout title="Contact" description="Contact the Gnet author">
      <header className="hero">
        <div className="container container--fluid">
          <h1>Contact</h1>
          <div className="hero--subtitle">
            Gnet is a open-source software developed by <a href="https://github.com/panjf2000">Andy Pan</a>. You can contact the author using any of the options below.
          </div>
        </div>
      </header>
      <main>
        <section>
          <div className="container">
            <div className="row">
              <div className="col">
                <a href="mailto:panjf2000@gmail.com" className="panel text--center">
                  <div className="panel--icon">
                    <i className="feather icon-mail"></i>
                  </div>
                  <div className="panel--title">panjf2000@gmail.com</div>
                  <div className="panel--description">Shoot me an email</div>
                </a>
              </div>
              <div className="col">
                <a href="https://twitter.com/_andy_pan" target="_blank" className="panel text--center">
                  <div className="panel--icon">
                    <i className="feather icon-twitter"></i>
                  </div>
                  <div className="panel--title">@_andy_pan</div>
                  <div className="panel--description">
                    Tweet at me 
                  </div>
                </a>
              </div>
              <div className="col">
                <a href="https://github.com/panjf2000/gnet/issues" target="_blank" className="panel text--center">
                  <div className="panel--icon">
                    <i className="feather icon-github"></i>
                  </div>
                  <div className="panel--title">Github issue</div>
                  <div className="panel--description">Create an issue</div>
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
