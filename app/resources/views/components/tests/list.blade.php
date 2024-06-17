@props(['select' => false, 'link' => true, 'tests' => []])

<table
    @class([
        'select' => $select
    ])>
    <thead>
    <tr>
       @if ($select)
        <th>
            
        </th>
        @endif
        <th>Test</th>
        <th>Type</th>
        @if (!$select)
        <th>Last result</th>
        <th>Error</th>
        @endif
    </tr>
    </thead>
    <tbody>
    @foreach($tests as $test)
    <tr>
        @if ($select)
        <td>
            <label for="selector-test-{{ $test->id }}" class="checkbox">
            <input type="checkbox" name="manual" id="selector-test-{{ $test->id }}">
            <span class="checkmark"></span>
            </label>
        </td>
        @endif
        <td>
            @if ($link)
                <a href="{{ $test->url }}">{{ $test->title }}</a>
            @else
                <span>{{$test->title}}</span>
            @endif
        </td>
        <td>
            @if ($test->type === \App\Enums\TestTypeEnum::automated->value)
                <span>Automated</span>
            @else
                <span>Manual</span>
            @endif
        </td>
        @if (!$select)
        <td>
            <span class="pass">1</span>
        </td>
        <td>1</td>
        @endif
    </tr>
  @endforeach
    </tbody>
</table>
