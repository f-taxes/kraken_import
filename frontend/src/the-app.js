/**
@license
Copyright (c) 2024 trading_peter
This program is available under Apache License Version 2.0
*/

import '@tp/tp-router/tp-router.js';
import './the-config.js';
import { LitElement, html, css } from 'lit';
import { fetchMixin } from '@tp/helpers/fetch-mixin.js';
import theme from './styles/theme.js';
import shared from './styles/shared.js';

class TheApp extends fetchMixin(LitElement) {
  static get styles() {
    return [
      theme,
      shared,
      css`
        :host {
          display: block;
          position: absolute;
          inset: 0;
        }

        .main {
          position: absolute;
          inset: 0;
          display: flex;
          flex-direction: column;
        }
      `
    ];
  }

  render() {
    const { routeParams, settings } = this;
    const p = routeParams || [];
    const page = p[0];

    return html`
      <tp-router @data-changed=${this.routeDataChanged}>
        <tp-route path="/config" data="config"></tp-route>
      </tp-router>
      
      <div class="main">
        ${page === 'config' ? html`<the-config .settings=${settings || {}}></the-config>` : null }
      </div>
    `;
  }

  static get properties() {
    return {
      // Data of the currently active route. Set by the router.
      route: { type: String, },

      // Params of the currently active route. Set by the router.
      routeParams: { type: Object },

      settings: { type: Object },
    };
  }

  routeDataChanged(e) {
    this.route = e.detail;
    this.routeParams = this.route.split('-');
  }

  connectedCallback() {
    super.connectedCallback();
    this.fetchSettings();
  }

  async fetchSettings() {
    const resp = this.get('/settings');

    if (resp.result) {
      this.settings = resp.data;
    }
  }
}

window.customElements.define('the-app', TheApp);