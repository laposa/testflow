@props(['suites' => []])
<table @class([])>
    <thead>
    <tr>
        <th>Test suite</th>
        <th></th>
    </tr>
    </thead>
    <tbody>
        @foreach($suites as $suite)
        <tr>
            <td>{{$suite['path']}}</td>
            <td><span class="expand" data-target="selector-suite-tests-{{ $suite['repository_id'] }}">Expand</span></td>
        </tr>
        <tr id="selector-suite-tests-{{ $suite['repository_id'] }}" class="collapsible">
            <td>
            </td>
        </tr>
        @endforeach
    </tbody>
</table>