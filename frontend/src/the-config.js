/**
@license
Copyright (c) 2024 trading_peter
This program is available under Apache License Version 2.0
*/

import '@tp/tp-form/tp-form.js';
import '@tp/tp-input/tp-input.js';
import '@tp/tp-button/tp-button.js';
import '@tp/tp-dialog/tp-dialog.js';
import './elements/card-box.js';
import { LitElement, html, css } from 'lit';
import icons from './icons';
import { formatTs, isZero } from './helpers/time.js';
import { fetchMixin } from '@tp/helpers/fetch-mixin.js';
import { DomQuery } from '@tp/helpers/dom-query.js';

class TheConfig extends fetchMixin(DomQuery(LitElement)) {
  static get styles() {
    return [
      css`
        :host {
          display: block;
          flex: 1;
          padding: 20px;
        }

        h2 {
          font-weight: normal;
          font-size: 22px;
        }

        h3 {
          font-weight: normal;
          font-size: 18px;
          margin: 0;
          padding: 0;
        }

        tp-button tp-icon {
          margin-left: 10px;
          --tp-icon-height: 18px;
          --tp-icon-width: 18px;
          --tp-icon-color: var(--text);
        }

        tp-button:hover tp-icon,
        tp-button:focus tp-icon {
          --tp-icon-color: var(--text-dark);
        }

        header {
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        card-box {
          max-width: 800px;
          margin: auto;
        }

        .empty {
          padding: 40px;
          text-align: center;
          font-size: 20px;
        }

        .src {
          display: grid;
          grid-template-columns: auto 1fr;
          grid-template-rows: 1fr 1fr auto;
          gap: 0 20px;
          grid-auto-flow: row;
          grid-template-areas:
            "logo label actions"
            "logo key actions";
          margin-top: 20px;
          background: var(--bg0);
          padding: 10px;
          border-radius: 4px;
        }

        .logo {
          padding: 10px;
          font-size: 30px;
          grid-area: logo;
          display: flex;
          justify-content: center;
          align-items: center;
          border-radius: 8px;
          background: var(--white);
        }

        .logo img {
          max-width: 80px;
        }

        .label {
          grid-area: label;
          align-items: center;
        }

        .key {
          grid-area: key;
        }

        .label,
        .key {
          display: grid;
          grid-template-columns: 1fr 1fr;
          grid-column-gap: 20px;
        }

        .label > div,
        .key > div {
          display: flex;
          align-items: center;
        }

        .actions {
          display: flex;
          flex-direction: row;
          justify-content: center;
          align-items: center;
          grid-area: actions;
        }

        .actions > * + * {
          margin-left: 10px;
        }

        .src label {
          color: var(--text-low);
          margin-right: 10px;
        }

        tp-form label {
          display: block;
          margin-bottom: 5px;
        }

        tp-input::part(wrap) {
          font-size: 18px;
          background: var(--input-bg);
          border: var(--input-border);
        }

        tp-input[focused]::part(wrap) {
          border: solid 1px var(--hl1);
        }

        tp-input[invalid]::part(wrap) {
          border: solid 1px var(--red);
        }

        tp-input::part(error-message) {
          font-size: 14px;
        }

        textarea {
          width: 100%;
          box-sizing: border-box;
          background: var(--input-bg);
          border: var(--input-border);
          outline: none;
          border-radius: 2px;
          color: var(--text);
          font-size: 18px;
          font-family: 'Source Sans Pro';
          height: 80px;
        }

        textarea:focus {
          border: solid 1px var(--hl1);
        }

        tp-form tp-input {
          margin-bottom: 20px;
        }

        .buttons-justified {
          margin-top: 30px;
          display: flex;
          justify-content: space-between;
        }

        tp-dialog h2 {
          margin: 0 0 20px 0;
        }

        tp-button.only-icon tp-icon {
          margin: 0;
          --tp-icon-height: 24px;
          --tp-icon-width: 24px;
        }
      `
    ];
  }

  render() {
    const { accounts, settings } = this;

    return html`
      <card-box>
        <h2>Add your Kraken accounts here</h2>
        <header>
          <h3>You have ${accounts.length} accounts connected.</h3>
          <tp-button @click=${this.startAddAccount}>Add <tp-icon .icon=${icons.add}></tp-icon></tp-button>
        </header>
        <div class="list">
          ${accounts.length == 0 ? html`
            <div class="empty">Click the "Add"-Button on the top right to add your first account</div>
          ` : accounts.map(con => html`
            <div class="src">
              <div class="logo">
                <img src="https://pro.kraken.com/app/favicon.ico"></img>
              </div>
              <div class="label">
                <div><label>Label:</label>${con.label}</div>
                <div><label>Last Fetched:</label>${con.lastFetched === '' || isZero(con.lastFetched) ? 'Never' : formatTs(con.lastFetched, settings?.dateTimeFormat)}</div>
              </div>
              <div class="key">
                <div><label>Api Key:</label>${con.key.substring(0, 6)}...</div>
                <div><label>Api Secret:</label>***</div>
              </div>
              <div class="actions">
                <tp-tooltip-wrapper text="Fetch newest data from this source" tooltipValign="top">
                  <tp-button id=${'fetch_' + con._id} class="only-icon" extended @click=${e => this.fetchData(e, con)}><tp-icon .icon=${icons.refresh}></tp-icon></tp-button>
                </tp-tooltip-wrapper>

                <tp-tooltip-wrapper text="Remove source and it's associated data" tooltipValign="top">
                  <tp-button class="only-icon" extended @click=${() => this.confirmRemoveAccount(con)}><tp-icon .icon=${icons.delete}></tp-icon></tp-button>
                </tp-tooltip-wrapper>
              </div>
            </div>
          `)}
        </div>
      </card-box>

      <tp-dialog id="addAccountDialog" showClose>
        <h2>Add Kraken account</h2>
        <tp-form @submit=${this.addAccount}>
          <form>
            <label>Label</label>
            <tp-input name="label" required errorMessage="Required">
              <input type="text">
            </tp-input>

            <label>API Key</label>
            <tp-input name="key" required errorMessage="Required">
              <input type="text">
            </tp-input>

            <label>API Secret</label>
            <tp-input name="secret" required errorMessage="Required">
              <input type="password">
            </tp-input>

            <label>Notes</label>
            <textarea name="notes"></textarea>

            <div class="buttons-justified">
              <tp-button dialog-dismiss>Cancel</tp-button>
              <tp-button id="addAccountBtn" submit>Add</tp-button>
            </div>
          </form>
        </tp-form>
      </tp-dialog>

      <tp-dialog id="removeAccountDialog" showClose>
        <h2>Confirm removal</h2>
        <p>Do you want to remove the Kraken account "${this.selAccount.label}"?<br>This will also delete all associated data like trades, transfers, etc.</p>
        <div class="buttons-justified">
          <tp-button dialog-dismiss>Cancel</tp-button>
          <tp-button class="danger" @click=${() => this.removeAccount()}>Yes, Remove</tp-button>
        </div>
      </tp-dialog>
    `;
  }

  static get properties() {
    return {
      items: { type: Array },
      active: { type: Boolean, reflect: true },
      accounts: { type: Array },
      settings: { type: Object },
      selAccount: { type: Object },
    };
  }

  constructor() {
    super();
    this.accounts = [];
    this.selAccount = {};
  }

  connectedCallback() {
    super.connectedCallback();
    this.fetchAccounts();
  }

  startAddAccount() {
    this.$.addAccountDialog.show();
  }

  async addAccount(e) {
    this.$.addAccountBtn.showSpinner();
    const resp = await this.post('/account/add', e.detail);
    
    if (resp.result) {
      this.$.addAccountBtn.showSuccess();
      this.$.addAccountDialog.close();
      this.fetchAccounts();
    } else {
      this.$.addAccountBtn.showError();
    }
  }

  async fetchAccounts() {
    const resp = await this.get('/account/list');

    if (resp.result) {
      this.accounts = resp.data;
    }
  }

  async fetchData(e, account) {
    const btn = e.target;
    btn.showSpinner();
    const resp = await this.post('/account/fetch/one', { id: account.id });
    if (!resp.result) {
      btn.showError();
    }
  }

  confirmRemoveAccount(account) {
    this.selAccount = account;
    this.$.removeAccountDialog.show();
  }

  removeAccount() {
    this.post('/account/remove', { id: this.selAccount.id });
    this.$.removeAccountDialog.close();
    this.fetchAccounts();
  }
}

window.customElements.define('the-config', TheConfig);