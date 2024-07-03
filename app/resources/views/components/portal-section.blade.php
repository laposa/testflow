@props(['border' => false, 'title' => ''])

<div @class(['portal-section', 'no-border' => !$border])>
    <h2>{{ $title }}</h2>
    {{ $slot }}
    <hr>
</div>
