<script lang="ts">
  export let onClick: () => void = () => null;
  export let text: string;
  export let disabled = false;
  export let submit = false;
</script>

<input
  type={submit ? 'submit' : 'button'}
  on:click={onClick}
  {disabled}
  value={text}
/>

<style lang="scss">
  @import '../colors';
  input {
    word-break: break-all;
    white-space: normal;
    min-width: 130px;
    flex-grow: 1;
    background-color: rgba($color: #000000, $alpha: 0);
    border: none;
    color: $text-color;
    text-transform: uppercase;
    padding: 15px 32px;
    font-weight: 700;
    text-align: center;
    text-decoration: none;
    display: inline-block;
    font-size: 16px;
    margin: 0.5em;
    border-style: solid;
    border-color: $element-border-color;
    border-radius: 0px;
    transition: all 0.5s cubic-bezier(0.16, 1, 0.3, 1);
    transform: skewX(10deg);
    margin: 5px;
    box-sizing: border-box;
  }
  input:hover:enabled,
  :focus {
    background-color: $element-bg-hover;
    color: $text-color-hover;
  }
  input:hover {
    cursor: pointer;
  }
  input:active {
    background-color: $element-bg-active !important;
  }
  input:disabled {
    cursor: not-allowed;
    color: $element-main-disabled;
    border-color: $element-main-disabled;
  }
</style>
