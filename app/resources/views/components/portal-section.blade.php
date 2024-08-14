@props(['border' => false, 'title' => ''])

<section @class(['portal-section', 'no-border' => !$border])>
    <h2>{{ $title }}</h2>
    {{ $slot }}
    <hr>
</section>
