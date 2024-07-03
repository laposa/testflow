@php
    /** @var \App\Models\SessionRun $session */
@endphp

<div @if ($pollingEnabled) wire:poll.2.5s @endif>
    <x-runs.list :runs="$session->runs" />
    
    <button type="submit" wire:click="createRun" wire:loading.attr="disabled" class="filled">Run all tests</button>
</div>
