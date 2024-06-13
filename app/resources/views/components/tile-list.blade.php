@props(['tiles' => []])

<div class="tiles">
    @foreach($tiles as $tile)
        <a href="{{ $tile['link'] }}" class="tiles__item">
            <strong>{{$tile['title']}}</strong>
            {{$tile['description']}}
        </a>
    @endforeach
</div>
