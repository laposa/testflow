@php
    /** @var \App\Models\Session $session */
    
@endphp

<div @if ($pollingEnabled) wire:poll.2.5s @endif>
    <x-runs.list :runs="$session->runs" :sessionId="$session->id" />
    {{ $session->name }}
    
    <button type="submit"
            wire:click="createRun"
            wire:loading.attr="disabled"
            @if (!$session->canRunTests())disabled @endif
            class="filled">Run all tests</button>
</div>
