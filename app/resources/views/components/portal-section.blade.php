@props(['border' => false, 'title' => '', 'width' => ''])

<div @class(['portal-section', 'section', 'no-border' => !$border, $width])>
    <h2>{{ $title }}</h2>
    {{ $slot }}
    <hr>
</div>
