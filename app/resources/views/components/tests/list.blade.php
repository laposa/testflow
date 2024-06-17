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
        <th>Last run</th>
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
            @switch($test->lastResult)
                @case(\App\Enums\TestResultEnum::pass->value)
                    <span class="pass">Passed</span>
                    @break
                @case(\App\Enums\TestResultEnum::fail->value)
                    <span class="fail">Failed</span>
                    @break
                @case(\App\Enums\TestResultEnum::notRun->value)
                    <span class="fresh">Not Run</span>
                    @break
            @endswitch
        </td>
        <td>{{$test->error}}</td>
        <td>
            @if ($test->timestamp)
                <a href={"sessions/1/test/{{ $test->id }}/run/{{ $test->id }}">
                    {{ now()->format('Y-m-d H:i:s') }}
                </a>
            @else

            @endif
        </td>
        @endif
    </tr>
  @endforeach
    </tbody>
</table>
