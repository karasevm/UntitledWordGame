<script lang="ts">
  export let value: string;
</script>

<input maxlength="30" type="text" bind:value />

<style lang="scss">
  @import '../colors';
  input {
    flex-grow: 1;
    background-color: $element-bg;
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
    transition: background-color 0.5s cubic-bezier(0.16, 1, 0.3, 1);
    transform: skewX(10deg);
    margin: 5px;
    box-sizing: border-box;
  }

  input:hover:enabled {
    background-color: $element-bg-hover;
    color: $text-color-hover;
  }
  input:focus {
    background-color: $element-bg-active !important;
    color: $text-color-hover;
  }
</style>
